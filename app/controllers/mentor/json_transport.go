package mentor

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"github.com/JorgeIba/Hurion/app/controllers/interactiontransport"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

// maxJSONFileBytes bounds the batch file download independently of the
// operation limit. Five MiB accommodates formatted JSON while bounding memory.
const maxJSONFileBytes = 5 * 1024 * 1024

const jsonFileExtension = ".json"

type jsonTransport struct {
	downloader *interactiontransport.AttachmentDownloader
}

func newJSONTransport(client *http.Client) *jsonTransport {
	return &jsonTransport{downloader: interactiontransport.NewAttachmentDownloader(client)}
}

// read uses the uploaded file when present; failed file reads never fall back to text.
func (reader *jsonTransport) read(source interactiontransport.JSONModalSource) (jsontext.Value, error) {
	content := source.Text
	if source.Attachment != nil {
		var err error
		content, err = reader.readFile(source.Attachment)
		if err != nil {
			return nil, err
		}
	}
	return parseJSONDocument(content)
}

type jsonDocumentError struct {
	line, column int
	err          error
}

func (err *jsonDocumentError) Error() string { return err.err.Error() }
func (err *jsonDocumentError) Unwrap() error { return err.err }

func parseJSONDocument(input string) (jsontext.Value, error) {
	var document jsontext.Value
	// Defer Unicode and duplicate-name checks to entry decoding,
	// which can report issues for multiple entries together.
	if err := jsonv2.Unmarshal([]byte(input), &document, jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true)); err != nil {
		location := &jsonDocumentError{err: fmt.Errorf("expected one JSON document: %w", err)}
		if syntax, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
			offset := max(0, min(len(input), int(syntax.ByteOffset)))
			before := input[:offset]
			location.line = strings.Count(before, "\n") + 1
			location.column = utf8.RuneCountInString(before[strings.LastIndex(before, "\n")+1:]) + 1
		}
		return nil, location
	}
	return document, nil
}

// readFile validates the attachment and downloads its complete contents.
// read checks document syntax; batch input decoding checks individual entries.
func (reader *jsonTransport) readFile(attachment *discordgo.MessageAttachment) (string, error) {
	if attachment == nil || !strings.EqualFold(filepath.Ext(attachment.Filename), jsonFileExtension) {
		return "", fmt.Errorf("upload a file with the %s extension", jsonFileExtension)
	}

	size := attachment.Size
	if size <= 0 || size > maxJSONFileBytes {
		return "", fmt.Errorf("the JSON file must be non-empty and at most %g MiB", float64(maxJSONFileBytes)/(1024*1024))
	}

	contents, err := reader.downloader.Download(attachment, maxJSONFileBytes)
	if err != nil {
		return "", err
	}
	if len(contents) != size {
		return "", fmt.Errorf("could not read the complete JSON file; upload it again")
	}

	return string(contents), nil
}
