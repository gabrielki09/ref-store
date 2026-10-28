package helpers

import (
	"ref-store/logger"
)

func CreateInitFolders(extraPaths ...string) (string, string, error) {
	storageDir, err := CreateInformedFolder(StorageDir)
	if err != nil {
		logger.Error("Erro ao criar diretório de armazenamento:", "err", err)
		return "", "", err
	}

	manifestDir, err := CreateInformedFolder(ManifestDir)
	if err != nil {
		logger.Error("Erro ao criar diretório de armazenamento do manifesto:", "err", err)
		return "", "", err
	}

	if len(extraPaths) > 0 {
		for _, path := range extraPaths {
			if _, err := CreateInformedFolder(path); err != nil {
				logger.Error("Erro ao criar diretório extra:", "err", err, "path", path)
				return "", "", err
			}
		}
	}

	return storageDir, manifestDir, nil
}
