package interactiontransport

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	maxAttachmentRedirects    = 3
	attachmentDownloadTimeout = 15 * time.Second

	discordAttachmentScheme         = "https"
	discordCDNHost                  = "cdn.discordapp.com"
	discordMediaHost                = "media.discordapp.net"
	discordAttachmentPathPrefix     = "/attachments/"
	discordEphemeralPathPrefix      = "/ephemeral-attachments/"
	messageAttachmentRedirectLimit  = "attachment download exceeded %d redirects"
	messageAttachmentInvalidURL     = "invalid Discord attachment URL"
	messageAttachmentInvalidSize    = "invalid attachment size or download limit"
	messageAttachmentDownloadFailed = "could not download the attachment: %w"
	messageAttachmentHTTPFailed     = "could not download the attachment (HTTP %d)"
	messageAttachmentReadFailed     = "could not read the attachment: %w"
	messageAttachmentEmpty          = "the attachment is empty"
)

var errAttachmentTooLarge = errors.New("the attachment exceeds the download limit")

// AttachmentDownloader applies one download policy to Discord attachments.
type AttachmentDownloader struct {
	client *http.Client
}

func NewAttachmentDownloader(client *http.Client) *AttachmentDownloader {
	if client == nil {
		client = &http.Client{}
	}
	boundedClient := *client
	boundedClient.Timeout = attachmentDownloadTimeout
	boundedClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) > maxAttachmentRedirects {
			return fmt.Errorf(messageAttachmentRedirectLimit, maxAttachmentRedirects)
		}
		return validateAttachmentURL(request.URL)
	}

	return &AttachmentDownloader{client: &boundedClient}
}

func validateAttachmentURL(location *url.URL) error {
	if location.Scheme != discordAttachmentScheme || location.User != nil ||
		(location.Host != discordCDNHost && location.Host != discordMediaHost) ||
		(!strings.HasPrefix(location.Path, discordAttachmentPathPrefix) && !strings.HasPrefix(location.Path, discordEphemeralPathPrefix)) {
		return fmt.Errorf(messageAttachmentInvalidURL)
	}

	return nil
}

// Download preserves the supplied URL, including its signature parameters.
// Format-specific validation and declared-size equality belong to callers.
func (downloader *AttachmentDownloader) Download(attachment *discordgo.MessageAttachment, maxBytes int64) ([]byte, error) {
	if attachment == nil || maxBytes <= 0 || attachment.Size < 0 || int64(attachment.Size) > maxBytes {
		return nil, fmt.Errorf(messageAttachmentInvalidSize)
	}

	location, err := url.Parse(attachment.URL)
	if err != nil {
		return nil, fmt.Errorf(messageAttachmentInvalidURL)
	}

	if err := validateAttachmentURL(location); err != nil {
		return nil, err
	}

	response, err := downloader.client.Get(attachment.URL)
	if err != nil {
		return nil, fmt.Errorf(messageAttachmentDownloadFailed, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(messageAttachmentHTTPFailed, response.StatusCode)
	}

	contents, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf(messageAttachmentReadFailed, err)
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf(messageAttachmentEmpty)
	}

	if int64(len(contents)) > maxBytes {
		return nil, fmt.Errorf("%w of %d bytes", errAttachmentTooLarge, maxBytes)
	}

	return contents, nil
}
