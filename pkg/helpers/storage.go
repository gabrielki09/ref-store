package helpers

import (
	"errors"
	"os"
	"path/filepath"
)

func StoreBlock(storageDir string, hash [32]byte, content []byte) error {
	if ByteToHash(content) != hash {
		return errors.New("conteudo diferente do hash calculado")
	}

	fileName := HexToString(hash[:])
	fileStoragePath := filepath.Join(storageDir, fileName)

	_, err := os.Stat(fileStoragePath)

	if err == nil {
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.WriteFile(fileStoragePath, content, 0666); err != nil {
		return err
	}

	return nil
}

func LoadBlock(storageDir string, hash [32]byte) ([]byte, error) {
	fileName := HexToString(hash[:])
	fileStoragePath := filepath.Join(storageDir, fileName)

	content, err := os.ReadFile(fileStoragePath)
	if err != nil {
		return nil, err
	}

	if ByteToHash(content) != hash {
		return nil, errors.New("conteudo diferente do hash calculado")
	}

	return content, nil
}
