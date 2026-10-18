package interactiontransport

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
)

func TestACProposalImageEnforcesTwentyMiBLimit(t *testing.T) {
	const limit = 20 * 1024 * 1024
	for _, test := range []struct {
		name     string
		declared int
		actual   int
	}{
		{name: "exact limit", declared: limit, actual: limit},
		{name: "declared over limit", declared: limit + 1, actual: limit + 1},
		{name: "body over limit", declared: limit, actual: limit + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: attachmentDownloadTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				body := bytes.Repeat([]byte{0}, test.actual)
				copy(body, "\x89PNG\r\n\x1a\n")
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
					Request:    request,
				}, nil
			})}
			image, filename, contentType, err := DownloadACProposalImage(NewAttachmentDownloader(client), &discordgo.MessageAttachment{
				URL:  "https://cdn.discordapp.com/attachments/1/2/proof.png",
				Size: test.declared,
			})
			if test.declared > limit {
				require.ErrorContains(t, err, "20 MiB")
				require.Zero(t, calls, "reject declared oversize before downloading")
			} else if test.actual > limit {
				require.EqualError(t, err, "The screenshot must be an image of 20 MiB or less.")
				require.Equal(t, 1, calls)
			} else {
				require.NoError(t, err)
				require.Equal(t, 1, calls)
				require.Len(t, image, limit)
				require.Equal(t, "ac-proposal-capture.png", filename)
				require.Equal(t, "image/png", contentType)
				return
			}
			require.Empty(t, image)
			require.Empty(t, filename)
			require.Empty(t, contentType)
		})
	}
}

func TestACProposalImageUsesSharedDownloadAndKeepsFormatValidation(t *testing.T) {
	for _, test := range []struct {
		name        string
		body        string
		extension   string
		contentType string
	}{
		{name: "PNG", body: "\x89PNG\r\n\x1a\n", extension: ".png", contentType: "image/png"},
		{name: "JPEG", body: "\xff\xd8\xff\xe0", extension: ".jpg", contentType: "image/jpeg"},
		{name: "GIF", body: "GIF89a", extension: ".gif", contentType: "image/gif"},
		{name: "WebP", body: "RIFF\x00\x00\x00\x00WEBPVP8 ", extension: ".webp", contentType: "image/webp"},
		{name: "non-image", body: "[]"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			client := &http.Client{Transport: attachmentDownloadTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return &http.Response{
						StatusCode: http.StatusFound,
						Header:     http.Header{"Location": []string{"https://media.discordapp.net/attachments/1/2/proof"}},
						Body:       io.NopCloser(strings.NewReader("")),
						Request:    request,
					}, nil
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(test.body)),
					Request:    request,
				}, nil
			})}
			image, filename, contentType, err := DownloadACProposalImage(NewAttachmentDownloader(client), &discordgo.MessageAttachment{
				URL:  "https://cdn.discordapp.com/attachments/1/2/proof?ex=123&hm=signed",
				Size: len(test.body),
			})
			require.Equal(t, 2, calls)
			if test.extension == "" {
				require.ErrorContains(t, err, "PNG, JPEG, GIF or WebP")
				require.Empty(t, image)
				require.Empty(t, filename)
				require.Empty(t, contentType)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.body, string(image))
				require.Equal(t, "ac-proposal-capture"+test.extension, filename)
				require.Equal(t, test.contentType, contentType)
			}
		})
	}
}
