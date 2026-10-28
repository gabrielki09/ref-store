package helpers

import (
	"os"
	"ref-store/logger"
)

const (
	StorageDir  = "storage"
	ManifestDir = "manifests"
	RestoredDir = "restored"
)

func CreateInformedFolder(path string) (string, error) {
	//fullPath := filepath.Join(path)

	if err := os.MkdirAll(path, 0755); err != nil {
		logger.Error("Erro ao criar a pasta informada:", "err", err)
		return "", err
	}

	return path, nil
}
