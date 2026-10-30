package helpers

import (
	"errors"
	"os"
	"path/filepath"
)

func StoreBlock(storageDir string, hash [32]byte, content []byte) (bool, error) {
	if ByteToHash(content) != hash {
		return false, errors.New("conteudo diferente do hash calculado")
	}

	fileName := HexToString(hash[:])
	fileStoragePath := filepath.Join(storageDir, fileName)

	_, err := os.Stat(fileStoragePath)

	if err == nil {
		return false, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	if err := os.WriteFile(fileStoragePath, content, 0666); err != nil {
		return false, err
	}

	return true, nil
}

func LoadBlock(storageDir string, hash [32]byte) ([]byte, error) {
	fileName := HexToString(hash[:])
	fileStoragePath := filepath.Join(storageDir, fileName)

	content, err := os.ReadFile(fileStoragePath)
	if err != nil {
		return nil, err
	}

	if ByteToHash(content) != hash {
		return nil, errors.New("conteúdo diferente do hash calculado")
	}

	return content, nil
}
