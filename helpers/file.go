package helpers

import (
	"os"
	"path/filepath"
	"ref-store/logger"
)

const (
	StorageDir  = "storage"
	ManifestDir = "manifests"
	RestoredDir = "restored"
)

func CreateInformedFolder(wd, path string) (string, error) {
	fullPath := filepath.Join(wd, path)

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		logger.Error("Erro ao criar a pasta informada:", "err", err)
		return "", err
	}

	return fullPath, nil
}
