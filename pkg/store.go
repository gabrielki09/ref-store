package pkg

import (
	"os"
	"path/filepath"
	"ref-store/logger"
	"ref-store/pkg/helpers"
)

func Store(filePath, storageDir, manifestDir string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("Erro ao abrir o arquivo:", "err", err)
		return "", err
	}

	defer file.Close()

	processResult, err := helpers.ProcessByte(file, storageDir)
	if err != nil {
		logger.Error("Erro ao processar os bytes do arquivo:", "err", err)
		return "", err
	}

	logger.Info(
		"Arquivo processado",
		"bytes", processResult.TotalBytes,
		"blocos", processResult.TotalBlock,
		"unicos", processResult.TotalUniqueBlocks,
		"repetidos", processResult.TotalRepeatedBlocks,
	)

	reconstructedJSON, err := helpers.BuildManifestJSON(
		processResult.BlockSequence,
		uint64(processResult.TotalBytes),
		uint64(helpers.ChunkSize),
		filePath,
		helpers.HexToString(processResult.OriginalFileHash[:]),
	)

	if err != nil {
		logger.Error("Erro ao criar o json do manifesto", "err", err)
		return "", err
	}

	manifestPath, err := helpers.SaveManifest(manifestDir, reconstructedJSON)
	if err != nil {
		logger.Error("Erro ao salvar o arquivo do manifesto:", "err", err)
		return "", err
	}

	logger.Info("Caminho do manifesto:", "info", manifestPath)

	return filepath.Base(manifestPath), nil
}
