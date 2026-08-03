package storage

import (
	"os"

	"github.com/JorgeIba/Hurion/app/logger"
)

// NewStorageFromEnv returns AzureBlobStorage if Azure environment variables are set,
// falling back to DiskStorage otherwise.
func NewStorageFromEnv(defaultDiskDir string) BlobStorage {
	hasConnStr := os.Getenv("AZURE_STORAGE_CONNECTION_STRING") != ""
	hasAccountAndKey := os.Getenv("AZURE_STORAGE_ACCOUNT") != "" && os.Getenv("AZURE_STORAGE_KEY") != ""

	if hasConnStr || hasAccountAndKey {
		azureStore, err := NewAzureBlobStorageFromEnv()
		if err == nil {
			if logger.Log != nil {
				logger.Log.Infof("Using AzureBlobStorage provider for container: %s", azureStore.ContainerName())
			}
			return azureStore
		}
		if logger.Log != nil {
			logger.Log.Warnf("Failed to initialize AzureBlobStorage from env, falling back to DiskStorage: %v", err)
		}
	}

	if logger.Log != nil {
		logger.Log.Infof("Using DiskStorage provider for directory: %s", defaultDiskDir)
	}
	diskStore, err := NewDiskStorage(defaultDiskDir)
	if err != nil {
		if logger.Log != nil {
			logger.Log.Errorf("Failed to initialize DiskStorage: %v", err)
		}
		return nil
	}
	return diskStore
}
