package mentor

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/JorgeIba/Hurion/app/presentation/discord"
	"github.com/bwmarrin/discordgo"
)

// maxJSONFileBytes bounds the batch file download independently of the
// operation limit. Five MiB accommodates formatted JSON while bounding memory.
const maxJSONFileBytes = 5 * 1024 * 1024

const (
	jsonFileExtension           = ".json"
	discordCDNHost              = "cdn.discordapp.com"
	discordMediaHost            = "media.discordapp.net"
	discordAttachmentPathPrefix = "/attachments/"
	discordEphemeralPathPrefix  = "/ephemeral-attachments/"
)

type jsonFileTransport struct {
	client *http.Client
}

func newJSONFileTransport(client *http.Client) *jsonFileTransport {
	if client == nil {
		client = &http.Client{}
	}

	boundedClient := *client
	boundedClient.Timeout = 15 * time.Second
	// Attachment URLs resolve directly; never follow redirects to another origin.
	boundedClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	return &jsonFileTransport{client: &boundedClient}
}

// read uses the uploaded file when present; failed file reads never fall back to text.
func (reader *jsonFileTransport) read(source discord.JSONModalSource) (string, error) {
	if source.Attachment == nil {
		return source.Text, nil
	}

	return reader.readFile(source.Attachment)
}

// readFile validates the attachment and downloads its complete contents.
// JSON syntax, Unicode, and entry types are validated by the existing parsers.
func (reader *jsonFileTransport) readFile(attachment *discordgo.MessageAttachment) (string, error) {
	if attachment == nil || !strings.EqualFold(filepath.Ext(attachment.Filename), jsonFileExtension) {
		return "", fmt.Errorf("upload a file with the %s extension", jsonFileExtension)
	}

	rawURL, size := attachment.URL, attachment.Size
	if size <= 0 || size > maxJSONFileBytes {
		return "", fmt.Errorf("the JSON file must be non-empty and at most %g MiB", float64(maxJSONFileBytes)/(1024*1024))
	}

	location, err := url.Parse(rawURL)
	if err != nil || location.Scheme != "https" || location.User != nil || location.Port() != "" ||
		(location.Host != discordCDNHost && location.Host != discordMediaHost) ||
		(!strings.HasPrefix(location.Path, discordAttachmentPathPrefix) && !strings.HasPrefix(location.Path, discordEphemeralPathPrefix)) {
		return "", fmt.Errorf("invalid Discord attachment URL")
	}

	response, err := reader.client.Get(location.String())
	if err != nil {
		return "", fmt.Errorf("could not download the JSON file; upload it again")
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not download the JSON file (HTTP %d); upload it again", response.StatusCode)
	}

	contents, err := io.ReadAll(io.LimitReader(response.Body, int64(size)+1))
	if err != nil || len(contents) != size {
		return "", fmt.Errorf("could not read the complete JSON file; upload it again")
	}
	return string(contents), nil
}

func batchParseReceipt(count int, issues []batchInputError) string {
	locations := make([]discord.JSONInputIssue, len(issues))

	for index, issue := range issues {
		locations[index] = discord.JSONInputIssue{Entry: issue.entry, Line: issue.line, Column: issue.column, Message: issue.message}
	}

	return discord.FormatJSONParseResult(count, locations)
}
