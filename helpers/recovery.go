package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"ref-store/logger"
)

func SaveRestoredFile(outputPath string, content []byte) error {
	tempDir := filepath.Dir(outputPath)

	tempFile, err := os.CreateTemp(tempDir, "temp-*")
	if err != nil {
		logger.Error("Erro ao criar arquivo temporário", "err", err)
		return err
	}

	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			logger.Warn(
				"Falha ao remover arquivo temporário",
				"path", tempFile.Name(),
				"err", err,
			)
		}
	}()

	_, writeTempErr := tempFile.Write(content)
	closeTempErr := tempFile.Close()

	if writeTempErr != nil || closeTempErr != nil {
		logger.Error(
			"Erro ao realizar a escrita / fechamento do arquivo temporário",
			"writeTempErr", writeTempErr,
			"closeTempErr", closeTempErr,
		)

		return errors.Join(writeTempErr, closeTempErr)
	}

	if err := os.Link(tempFile.Name(), outputPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			savedFileContent, readErr := os.ReadFile(outputPath)
			if readErr != nil {
				logger.Error("Erro ao ler os dados do arquivo:", "readErr", readErr)
				return readErr
			}

			contentHash := ByteToHash(savedFileContent)

			if contentHash == ByteToHash(content) {
				return nil
			}

			return errors.New("arquivo existente possui conteúdo diferente")
		}

		logger.Error("Erro ao públicar o arquivo final", "err", err)
		return err
	}

	return nil
}
