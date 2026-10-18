package interactiontransport

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"
)

type attachmentDownloadTransport func(*http.Request) (*http.Response, error)

func (transport attachmentDownloadTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestAttachmentDownloadValidatesInitialAndRedirectDestinations(t *testing.T) {
	const signedURL = "https://cdn.discordapp.com/attachments/1/2/file?ex=123&is=456&hm=abc%2Bdef"
	for _, destination := range []string{
		"https://cdn.discordapp.com/attachments/1/2/file?ex=new&hm=signature",
		"https://media.discordapp.net/ephemeral-attachments/1/2/file",
		"/attachments/1/2/other?ex=new&hm=signature",
		"https://example.com/attachments/1/2/file",
		"https://cdn.discordapp.com.evil.example/attachments/1/2/file",
		"http://cdn.discordapp.com/attachments/1/2/file",
		"https://cdn.discordapp.com:8443/attachments/1/2/file",
		"https://cdn.discordapp.com:443/attachments/1/2/file",
		"https://user@cdn.discordapp.com/attachments/1/2/file",
		"https://cdn.discordapp.com/other/file",
	} {
		for _, redirected := range []bool{false, true} {
			name := "initial/" + destination
			if redirected {
				name = "redirect/" + destination
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				allowed := destination == "https://cdn.discordapp.com/attachments/1/2/file?ex=new&hm=signature" ||
					destination == "https://media.discordapp.net/ephemeral-attachments/1/2/file" ||
					(redirected && strings.HasPrefix(destination, "/attachments/"))
				calls := 0
				client := &http.Client{Transport: attachmentDownloadTransport(func(request *http.Request) (*http.Response, error) {
					calls++
					if redirected && calls == 1 {
						require.Equal(t, signedURL, request.URL.String(), "signature parameters must not be rewritten")
						return &http.Response{
							StatusCode: http.StatusFound,
							Header:     http.Header{"Location": []string{destination}},
							Body:       io.NopCloser(strings.NewReader("")),
							Request:    request,
						}, nil
					}
					require.True(t, allowed, "a rejected destination must never receive an HTTP request")
					deadline, bounded := request.Context().Deadline()
					require.True(t, bounded)
					require.Positive(t, time.Until(deadline))
					require.LessOrEqual(t, time.Until(deadline), 15*time.Second)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("proof")),
						Request:    request,
					}, nil
				})}
				location := destination
				if redirected {
					location = signedURL
				}
				contents, err := NewAttachmentDownloader(client).Download(&discordgo.MessageAttachment{
					URL:  location,
					Size: 5,
				}, 5)
				if allowed {
					require.NoError(t, err)
					require.Equal(t, "proof", string(contents))
				} else {
					require.ErrorContains(t, err, "URL")
					require.Empty(t, contents)
				}
				expectedCalls := 0
				if allowed {
					expectedCalls++
				}
				if redirected {
					expectedCalls++
				}
				require.Equal(t, expectedCalls, calls)
			})
		}
	}
}

func TestAttachmentDownloadAllowsThreeRedirectsAndStopsBeforeTheFourth(t *testing.T) {
	for _, redirects := range []int{3, 4} {
		t.Run(string(rune('0'+redirects))+" redirects", func(t *testing.T) {
			t.Parallel()
			calls := 0
			client := &http.Client{Transport: attachmentDownloadTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				status := http.StatusOK
				headers := make(http.Header)
				if calls <= redirects {
					status = http.StatusTemporaryRedirect
					headers.Set("Location", "https://media.discordapp.net/attachments/1/2/proof")
				}
				return &http.Response{
					StatusCode: status,
					Header:     headers,
					Body:       io.NopCloser(strings.NewReader("proof")),
					Request:    request,
				}, nil
			})}
			contents, err := NewAttachmentDownloader(client).Download(&discordgo.MessageAttachment{
				URL:  "https://cdn.discordapp.com/attachments/1/2/proof",
				Size: 5,
			}, 5)
			require.Equal(t, 4, calls, "the initial request plus at most three redirected requests")
			if redirects == 3 {
				require.NoError(t, err)
				require.Equal(t, "proof", string(contents))
			} else {
				require.ErrorContains(t, err, "exceeded 3 redirects")
				require.Empty(t, contents)
			}
		})
	}
}

type downloadBody struct {
	io.Reader
	closed bool
}

func (body *downloadBody) Close() error {
	body.closed = true
	return nil
}

type failedDownloadReader struct{}

func (failedDownloadReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestAttachmentDownloadBoundsAndClosesTheBody(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		body      io.Reader
		wantError string
	}{
		{name: "size boundary", status: 200, body: strings.NewReader("12345")},
		{name: "body exceeds limit", status: 200, body: strings.NewReader("123456789"), wantError: "exceeds"},
		{name: "empty body", status: 200, body: strings.NewReader(""), wantError: "empty"},
		{name: "read failure", status: 200, body: failedDownloadReader{}, wantError: "read failed"},
		{name: "HTTP failure", status: 403, body: strings.NewReader("denied"), wantError: "HTTP 403"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			body := &downloadBody{Reader: test.body}
			client := &http.Client{Transport: attachmentDownloadTransport(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: test.status,
					Body:       body,
					Request:    request,
				}, nil
			})}
			contents, err := NewAttachmentDownloader(client).Download(&discordgo.MessageAttachment{
				URL:  "https://cdn.discordapp.com/attachments/1/2/file",
				Size: 5,
			}, 5)
			require.True(t, body.closed)
			if test.wantError == "" {
				require.NoError(t, err)
				require.Equal(t, "12345", string(contents))
			} else {
				require.ErrorContains(t, err, test.wantError)
				require.Empty(t, contents)
				if test.name == "body exceeds limit" {
					require.Equal(t, 3, test.body.(*strings.Reader).Len(), "read at most limit + 1 bytes")
				}
			}
		})
	}
}

func TestAttachmentDownloaderDoesNotMutateItsInjectedClient(t *testing.T) {
	client := &http.Client{
		Timeout: time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("original policy")
		},
	}
	downloader := NewAttachmentDownloader(client)
	require.NotSame(t, client, downloader.client)
	require.Equal(t, time.Second, client.Timeout)
	require.ErrorContains(t, client.CheckRedirect(nil, nil), "original policy")
	require.Equal(t, 15*time.Second, downloader.client.Timeout)
}
