package pkg

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"ref-store/logger"
	"ref-store/pkg/helpers"
)

func Restore(
	manifestPath string,
	storageDir string,
	restoredDir string,
) error {
	tempFile, err := os.CreateTemp(restoredDir, "temp-*")
	if err != nil {
		logger.Error("Erro ao criar arquivo temporário", "err", err)
		return err
	}

	defer func() {
		_ = tempFile.Close()

		if err := os.Remove(tempFile.Name()); err != nil {
			logger.Warn(
				"Falha ao remover arquivo temporário",
				"path", tempFile.Name(),
				"err", err,
			)
		}
	}()

	manifestData, err := helpers.LoadManifest(manifestPath)
	if err != nil {
		logger.Error("Erro ao carregar o manifesto:", "err", err)
		return err
	}

	blocks, err := helpers.DecodeBlockSequence(manifestData.Blocks)
	if err != nil {
		return err
	}

	var totalBytes uint64
	hasher := helpers.CreateHasherSHA256()

	for _, hash := range blocks {
		content, err := helpers.LoadBlock(storageDir, hash)

		if err != nil {
			return err
		}

		if _, err := tempFile.Write(content); err != nil {
			return err
		}

		if _, err := hasher.Write(content); err != nil {
			return err
		}

		totalBytes += uint64(len(content))
	}

	calculatedHash := helpers.HexToString(hasher.Sum(nil))

	if totalBytes != manifestData.FileSize {
		return errors.New("arquivo inválido")
	}

	if calculatedHash != manifestData.FileHash {
		return errors.New("hash reconstruído diverge do manifesto")
	}

	fileName := manifestData.FileName

	if fileName == "" ||
		fileName == "." ||
		fileName == ".." ||
		filepath.Base(fileName) != fileName {
		return errors.New("nome inválido no manifesto")
	}

	outputPath := filepath.Join(restoredDir, fileName)

	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := publishRestoredFile(
		tempFile.Name(),
		outputPath,
		calculatedHash,
		totalBytes,
	); err != nil {
		return err
	}

	return nil
}

func publishRestoredFile(
	tempPath string,
	outputPath string,
	expectedHash string,
	expectedSize uint64,
) error {
	if err := os.Link(tempPath, outputPath); err != nil {
		if errors.Is(err, os.ErrExist) {

			/*savedFileContent, readErr := os.ReadFile(outputPath)
			if readErr != nil {
				logger.Error("Erro ao ler os dados do arquivo:", "readErr", readErr)
				return readErr
			}

			contentHash := helpers.ByteToHash(savedFileContent)

			if contentHash == helpers.ByteToHash(content) {
				return nil
			}*/

			existingFile, err := os.Open(outputPath)
			if err != nil {
				return err
			}
			defer existingFile.Close()

			existingHasher := helpers.CreateHasherSHA256()
			size, err := io.Copy(existingHasher, existingFile)
			if err != nil {
				return err
			}

			existingHash := helpers.HexToString(existingHasher.Sum(nil))

			if uint64(size) == expectedSize && existingHash == expectedHash {
				return nil
			}

			return errors.New("arquivo existente possui conteúdo diferente")
		}

		logger.Error("Erro ao públicar o arquivo final", "err", err)
		return err
	}

	return nil
}
