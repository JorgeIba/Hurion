package mentor

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

type attachmentRoundTripper func(*http.Request) (*http.Response, error)

func (transport attachmentRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestReadJSONFile(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, url, body, filename string
		size, status              int
		transportErr              error
		wantError                 string
		wantRequest               bool
	}{
		{name: "signed CDN URL", url: "https://cdn.discordapp.com/attachments/1/2/batch.json?ex=123&hm=signed", body: "[]", size: 2, status: 200, wantRequest: true},
		{name: "ephemeral file", url: "https://media.discordapp.net/ephemeral-attachments/1/2/batch.json", body: "[]", size: 2, status: 200, wantRequest: true},
		{name: "size boundary", body: strings.Repeat(" ", maxJSONFileBytes), size: maxJSONFileBytes, status: 200, wantRequest: true},
		{name: "over size limit", size: maxJSONFileBytes + 1, wantError: "5 MiB"},
		{name: "empty file", wantError: "non-empty"},
		{name: "uppercase extension", filename: "batch.JSON", body: "[]", size: 2, status: 200, wantRequest: true},
		{name: "wrong extension", filename: "batch.txt", body: "[]", size: 2, wantError: ".json extension"},
		{name: "negative size", size: -1, wantError: "non-empty"},
		{name: "untrusted host", url: "https://example.com/attachments/1/2/a.json", size: 2, wantError: "URL"},
		{name: "HTTP URL", url: "http://cdn.discordapp.com/attachments/1/2/a.json", size: 2, wantError: "URL"},
		{name: "unexpected port", url: "https://cdn.discordapp.com:8443/attachments/1/2/a.json", size: 2, wantError: "URL"},
		{name: "unexpected path", url: "https://cdn.discordapp.com/not-an-attachment", size: 2, wantError: "URL"},
		{name: "credentials", url: "https://user@cdn.discordapp.com/attachments/1/2/a.json", size: 2, wantError: "URL"},
		{name: "HTTP failure", size: 2, status: 403, wantRequest: true, wantError: "HTTP 403"},
		{name: "download failure", size: 2, transportErr: errors.New("offline"), wantRequest: true, wantError: "download"},
		{name: "truncated file", body: "[", size: 2, status: 200, wantRequest: true, wantError: "complete"},
		{name: "body exceeds declared size", body: "[]extra", size: 2, status: 200, wantRequest: true, wantError: "complete"},
		{name: "redirect rejected", size: 2, status: 302, wantRequest: true, wantError: "HTTP 302"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			location := test.url
			if location == "" {
				location = "https://cdn.discordapp.com/attachments/1/2/batch.json"
			}
			calls := 0
			client := &http.Client{Transport: attachmentRoundTripper(func(request *http.Request) (*http.Response, error) {
				calls++
				assert.Equal(t, location, request.URL.String())
				if test.transportErr != nil {
					return nil, test.transportErr
				}
				return &http.Response{StatusCode: test.status, Header: http.Header{"Location": []string{"https://example.com/file"}}, Body: io.NopCloser(strings.NewReader(test.body)), Request: request}, nil
			})}
			reader := newJSONTransport(client)
			filename := test.filename
			if filename == "" {
				filename = "batch.json"
			}
			contents, err := reader.readFile(&discordgo.MessageAttachment{Filename: filename, URL: location, Size: test.size})
			if test.wantError != "" {
				assert.ErrorContains(t, err, test.wantError)
				assert.Empty(t, contents)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.body, contents)
			}
			if test.wantRequest {
				assert.Equal(t, 1, calls)
			} else {
				assert.Zero(t, calls)
			}
			assert.Zero(t, client.Timeout, "constructor must not mutate the shared client")
			assert.Equal(t, 15*time.Second, reader.client.Timeout)
		})
	}
}

func TestBatchSyntaxErrorLocation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		input  string
		line   int
		column int
	}{
		{name: "CRLF and Unicode column", input: "[\r\n  {\"name\":\"á\",?}\r\n]", line: 2, column: 15},
		{name: "trailing comma", input: "[\n{},\n]", line: 3, column: 1},
		{name: "empty input", line: 1, column: 1},
		{name: "missing array closing bracket", input: "[{}", line: 1, column: 4},
		{name: "EOF after newline", input: "[{}\n", line: 2, column: 1},
		{name: "EOF after CRLF", input: "[{}\r\n", line: 2, column: 1},
		{name: "EOF after Unicode", input: "[\"á\"", line: 1, column: 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			document, err := parseJSONDocument(test.input)
			assert.Nil(t, document)
			var syntax *jsonDocumentError
			if assert.ErrorAs(t, err, &syntax) {
				assert.Equal(t, test.line, syntax.line)
				assert.Equal(t, test.column, syntax.column)
			}
		})
	}
}
