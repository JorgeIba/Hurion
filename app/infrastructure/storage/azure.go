package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

var (
	ErrUninitializedClient  = errors.New("azure storage: client is uninitialized")
	ErrMissingAccountName   = errors.New("azure storage: account name is required")
	ErrMissingCredentials   = errors.New("azure storage: missing account key or connection string")
	ErrInvalidContainerName = errors.New("azure storage: container name must be 3-63 characters, start and end with a lowercase letter or number, and contain only lowercase letters, numbers, or non-consecutive hyphens")
)

// containerNameRegex enforces Azure's container naming specification (3-63 chars, lowercase letters, numbers, hyphens).
// Must start and end with an alphanumeric character (e.g., "exports", "my-exports-1").
// Rejects uppercase ("EXPORTS"), underscores ("my_exports"), or consecutive hyphens ("my--exports").
var containerNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)

// AzureUploader abstracts SDK buffer uploads for dependency inversion and unit testing.
type AzureUploader interface {
	UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte, options *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error)
}

type AzureBlobStorage struct {
	accountName   string
	containerName string
	client        AzureUploader
}

// NewAzureBlobStorage constructs a validated AzureBlobStorage instance.
func NewAzureBlobStorage(accountName, containerName string, client AzureUploader) (*AzureBlobStorage, error) {
	if accountName == "" {
		return nil, ErrMissingAccountName
	}
	if client == nil {
		return nil, ErrUninitializedClient
	}

	validatedContainer, err := validateContainerName(containerName)
	if err != nil {
		return nil, err
	}

	return &AzureBlobStorage{
		accountName:   accountName,
		containerName: validatedContainer,
		client:        client,
	}, nil
}

func (s *AzureBlobStorage) AccountName() string {
	return s.accountName
}

func (s *AzureBlobStorage) ContainerName() string {
	return s.containerName
}

// NewAzureBlobStorageFromEnv initializes AzureBlobStorage using environment variables
func NewAzureBlobStorageFromEnv() (*AzureBlobStorage, error) {
	accountName := os.Getenv("AZURE_STORAGE_ACCOUNT")
	containerName := os.Getenv("AZURE_STORAGE_CONTAINER")
	accountKey := os.Getenv("AZURE_STORAGE_KEY")
	connStr := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")

	if accountName == "" {
		return nil, ErrMissingAccountName
	}
	if containerName == "" {
		containerName = "exports"
	}

	var client *azblob.Client
	var err error

	if connStr != "" {
		client, err = azblob.NewClientFromConnectionString(connStr, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure Blob client from connection string: %w", err)
		}
	} else if accountKey != "" {
		cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure Shared Key credential: %w", err)
		}
		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure Blob client: %w", err)
		}
	} else {
		return nil, ErrMissingCredentials
	}

	return NewAzureBlobStorage(accountName, containerName, client)
}

func (s *AzureBlobStorage) Save(ctx context.Context, name string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context canceled before Azure blob upload of %s: %w", name, err)
	}

	if s.client == nil {
		return ErrUninitializedClient
	}

	safeName, err := sanitizeFilename(name)
	if err != nil {
		return fmt.Errorf("invalid filename %q: %w", name, err)
	}
	_, err = s.client.UploadBuffer(ctx, s.containerName, safeName, data, nil)
	if err != nil {
		return fmt.Errorf("failed to upload blob %s to container %s: %w", safeName, s.containerName, err)
	}

	return nil
}

func validateContainerName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "exports"
	}

	if len(trimmed) < 3 || len(trimmed) > 63 {
		return "", ErrInvalidContainerName
	}

	if strings.Contains(trimmed, "--") {
		return "", ErrInvalidContainerName
	}

	if !containerNameRegex.MatchString(trimmed) {
		return "", ErrInvalidContainerName
	}

	return trimmed, nil
}
