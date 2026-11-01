package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"ref-store/logger"
)

func StoreBlock(
	storageDir string,
	hash [32]byte,
	content []byte,
) (bool, error) {
	if ByteToHash(content) != hash {
		return false, errors.New("conteudo diferente do hash calculado")
	}

	fileName := HexToString(hash[:])
	fileStoragePath := filepath.Join(storageDir, fileName)
	tempFile, err := os.CreateTemp(storageDir, fileName)

	if err != nil {
		return false, err
	}

	defer func() {
		_ = tempFile.Close()

		if err := os.Remove(tempFile.Name()); err != nil {
			logger.Warn("Falha ao remover temporário", "err", err)
		}
	}()

	if _, err := tempFile.Write(content); err != nil {
		return false, err
	}

	if err := tempFile.Close(); err != nil {
		return false, err
	}

	err = os.Link(tempFile.Name(), fileStoragePath)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrExist) {
		if _, err := LoadBlock(storageDir, hash); err != nil {
			return false, err
		}

		return false, nil
	}

	return false, nil
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
