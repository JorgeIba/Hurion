package interactiontransport

import (
	"errors"
	"net/http"

	"github.com/JorgeIba/Hurion/app/bot/discordcommands"
	"github.com/JorgeIba/Hurion/app/logger"
	"github.com/bwmarrin/discordgo"
)

const (
	maxACProposalImageSize = 20 * 1024 * 1024

	messageACProposalUsage                = "Use `/ac submit` with a problem ID and an optional screenshot."
	messageACProposalInvalidProblemID     = "Enter a valid positive problem ID."
	messageACProposalInvalidScreenshot    = "Please attach a valid screenshot."
	messageACProposalUnreadableScreenshot = "Could not read the screenshot. Please attach it again."
	messageACProposalMissingScreenshot    = "Could not find the screenshot. Please attach it again."
	messageACProposalImageSize            = "The screenshot must be an image of 20 MiB or less."
	messageACProposalUnsupportedImage     = "Please use a PNG, JPEG, GIF or WebP screenshot."
	acProposalImageBasename               = "ac-proposal-capture"
	imageMIMEGIF                          = "image/gif"
	imageMIMEJPEG                         = "image/jpeg"
	imageMIMEPNG                          = "image/png"
	imageMIMEWebP                         = "image/webp"
	imageExtensionGIF                     = ".gif"
	imageExtensionJPEG                    = ".jpg"
	imageExtensionPNG                     = ".png"
	imageExtensionWebP                    = ".webp"
)

// ParseACProposalOptions reads a submitted command and resolves its optional evidence.
func ParseACProposalOptions(data discordgo.ApplicationCommandInteractionData) (int64, *discordgo.MessageAttachment, error) {
	if len(data.Options) != 1 ||
		data.Options[0] == nil ||
		data.Options[0].Type != discordgo.ApplicationCommandOptionSubCommand ||
		data.Options[0].Name != discordcommands.ACSubmitCommandName {
		return 0, nil, errors.New(messageACProposalUsage)
	}

	var problemOption, screenshotOption *discordgo.ApplicationCommandInteractionDataOption
	for _, option := range data.Options[0].Options {
		if option == nil {
			continue
		}

		switch option.Name {
		case discordcommands.ACProblemIDOptionName:
			problemOption = option
		case discordcommands.ACScreenshotOptionName:
			screenshotOption = option
		}
	}

	if problemOption == nil || problemOption.Type != discordgo.ApplicationCommandOptionInteger {
		return 0, nil, errors.New(messageACProposalInvalidProblemID)
	}

	// DiscordGo decodes INTEGER option values as float64. Check the representation
	// before using its accessor, which panics on an unexpected value type.
	if _, ok := problemOption.Value.(float64); !ok {
		return 0, nil, errors.New(messageACProposalInvalidProblemID)
	}

	problemID := problemOption.IntValue()
	if problemID <= 0 {
		return 0, nil, errors.New(messageACProposalInvalidProblemID)
	}

	if screenshotOption == nil {
		return problemID, nil, nil
	}

	if screenshotOption.Type != discordgo.ApplicationCommandOptionAttachment {
		return 0, nil, errors.New(messageACProposalInvalidScreenshot)
	}

	attachmentID, ok := screenshotOption.Value.(string)
	if !ok || attachmentID == "" || data.Resolved == nil {
		return 0, nil, errors.New(messageACProposalUnreadableScreenshot)
	}

	attachment := data.Resolved.Attachments[attachmentID]
	if attachment == nil || attachment.ID != attachmentID {
		return 0, nil, errors.New(messageACProposalMissingScreenshot)
	}

	return problemID, attachment, nil
}

// DownloadACProposalImage prepares an optional screenshot for publication.
func DownloadACProposalImage(downloader *AttachmentDownloader, attachment *discordgo.MessageAttachment) ([]byte, string, string, error) {
	if attachment == nil || attachment.Size < 0 || attachment.Size > maxACProposalImageSize {
		return nil, "", "", errors.New(messageACProposalImageSize)
	}

	image, err := downloader.Download(attachment, maxACProposalImageSize)
	if err != nil {
		logger.Log.Warnf("AC proposal screenshot download failed: %v", err)
		if errors.Is(err, errAttachmentTooLarge) {
			return nil, "", "", errors.New(messageACProposalImageSize)
		}
		return nil, "", "", errors.New(messageACProposalUnreadableScreenshot)
	}

	contentType := http.DetectContentType(image)
	extension := map[string]string{
		imageMIMEGIF:  imageExtensionGIF,
		imageMIMEJPEG: imageExtensionJPEG,
		imageMIMEPNG:  imageExtensionPNG,
		imageMIMEWebP: imageExtensionWebP,
	}[contentType]

	if extension == "" {
		return nil, "", "", errors.New(messageACProposalUnsupportedImage)
	}

	return image, acProposalImageBasename + extension, contentType, nil
}
