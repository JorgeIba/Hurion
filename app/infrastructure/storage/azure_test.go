package storage

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAzureUploader mocks AzureUploader interface for fast, network-free unit testing
type mockAzureUploader struct {
	uploadFunc func(ctx context.Context, containerName string, blobName string, buffer []byte, options *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error)
}

func (m *mockAzureUploader) UploadBuffer(ctx context.Context, containerName string, blobName string, buffer []byte, options *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, containerName, blobName, buffer, options)
	}
	return azblob.UploadBufferResponse{}, nil
}

func TestNewAzureBlobStorage_Validation(t *testing.T) {
	t.Parallel()

	mockClient := &mockAzureUploader{}

	tests := []struct {
		name          string
		accountName   string
		containerName string
		client        AzureUploader
		expectedErr   error
	}{
		{
			name:          "missing account name",
			accountName:   "",
			containerName: "exports",
			client:        mockClient,
			expectedErr:   ErrMissingAccountName,
		},
		{
			name:          "uninitialized client",
			accountName:   "testaccount",
			containerName: "exports",
			client:        nil,
			expectedErr:   ErrUninitializedClient,
		},
		{
			name:          "default container fallback",
			accountName:   "testaccount",
			containerName: "",
			client:        mockClient,
			expectedErr:   nil,
		},
		{
			name:          "invalid container name uppercase",
			accountName:   "testaccount",
			containerName: "MY_EXPORTS",
			client:        mockClient,
			expectedErr:   ErrInvalidContainerName,
		},
		{
			name:          "invalid container name consecutive hyphens",
			accountName:   "testaccount",
			containerName: "my--exports",
			client:        mockClient,
			expectedErr:   ErrInvalidContainerName,
		},
		{
			name:          "invalid container name too short",
			accountName:   "testaccount",
			containerName: "ab",
			client:        mockClient,
			expectedErr:   ErrInvalidContainerName,
		},
		{
			name:          "valid hyphenated container name",
			accountName:   "testaccount",
			containerName: "my-valid-exports-1",
			client:        mockClient,
			expectedErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store, err := NewAzureBlobStorage(tt.accountName, tt.containerName, tt.client)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				expectedContainer := tt.containerName
				if expectedContainer == "" {
					expectedContainer = "exports"
				}
				assert.Equal(t, expectedContainer, store.ContainerName())
				assert.Equal(t, "testaccount", store.AccountName())
			}
		})
	}
}

func TestAzureBlobStorage_Save(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMock   func() AzureUploader
		ctx         func() (context.Context, context.CancelFunc)
		filename    string
		content     []byte
		expectedErr string
	}{
		{
			name: "successful blob upload",
			setupMock: func() AzureUploader {
				return &mockAzureUploader{
					uploadFunc: func(ctx context.Context, containerName string, blobName string, buffer []byte, options *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error) {
						assert.Equal(t, "exports", containerName)
						assert.Equal(t, "basicos_1_score.json", blobName)
						assert.Equal(t, []byte("{\"test\":true}"), buffer)
						return azblob.UploadBufferResponse{}, nil
					},
				}
			},
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			filename:    "basicos_1_score.json",
			content:     []byte("{\"test\":true}"),
			expectedErr: "",
		},
		{
			name: "upload network failure error",
			setupMock: func() AzureUploader {
				return &mockAzureUploader{
					uploadFunc: func(ctx context.Context, containerName string, blobName string, buffer []byte, options *azblob.UploadBufferOptions) (azblob.UploadBufferResponse, error) {
						return azblob.UploadBufferResponse{}, errors.New("network failure")
					},
				}
			},
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			filename:    "test.json",
			content:     []byte("{}"),
			expectedErr: "failed to upload blob",
		},
		{
			name: "canceled context error",
			setupMock: func() AzureUploader {
				return &mockAzureUploader{}
			},
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Pre-cancel context
				return ctx, cancel
			},
			filename:    "test.json",
			content:     []byte("{}"),
			expectedErr: "canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockClient := tt.setupMock()
			store, err := NewAzureBlobStorage("testaccount", "exports", mockClient)
			require.NoError(t, err)

			ctx, cancel := tt.ctx()
			defer cancel()

			err = store.Save(ctx, tt.filename, tt.content)
			if tt.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewAzureBlobStorageFromEnv_MissingAccountName_ReturnsError(t *testing.T) {
	os.Unsetenv("AZURE_STORAGE_ACCOUNT")
	os.Unsetenv("AZURE_STORAGE_KEY")
	os.Unsetenv("AZURE_STORAGE_CONNECTION_STRING")

	_, err := NewAzureBlobStorageFromEnv()
	require.ErrorIs(t, err, ErrMissingAccountName)
}

func TestNewStorageFromEnv_FallbackToDiskStorage(t *testing.T) {
	os.Unsetenv("AZURE_STORAGE_ACCOUNT")
	os.Unsetenv("AZURE_STORAGE_KEY")

	tempDir := t.TempDir()
	store := NewStorageFromEnv(tempDir)

	_, ok := store.(*DiskStorage)
	assert.True(t, ok, "should fallback to DiskStorage when Azure env vars are absent")
}
