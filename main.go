package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"ref-store/helpers"
	"ref-store/logger"
)

const ChunkSize = 4

type ProcessResult struct {
	storedHashes        StoredHashes
	blockSequence       [][32]byte
	originalFileHash    [32]byte
	totalBlock          int
	totalBytes          int
	totalRepeatedBlocks int
}

type StoredHashes map[[32]byte][]byte

func showStoredHashs(storedHashs StoredHashes) {
	logger.Info("Bytes armazenados:", "info", storedHashs)

	for _, v := range storedHashs {
		logger.Info("Byte:", "info", string(v))
	}
}

func processByte(file *os.File) (ProcessResult, error) {
	b := make([]byte, ChunkSize)

	storedHashs := make(StoredHashes)
	var blockSequence [][32]byte
	totalBlock := 0
	totalBytes := 0
	totalRepeatedBlocks := 0

	hasher := helpers.CreateHasherSHA256()

	for {
		n, err := io.ReadFull(file, b)

		if n > 0 {
			logger.Info("Bloco lido:", "conteudo", string(b[:n]), "n", n)
			totalBlock += 1
			totalBytes += n

			hasher.Write(b[:n])
			hash := helpers.ByteToHash(b[:n])

			logger.Info("Hash gerado: ", "hash", hash)

			_, ok := storedHashs[hash]

			blockSequence = append(blockSequence, hash)

			if !ok {
				newSlice := make([]byte, n)

				copy(newSlice, b[:n])

				storedHashs[hash] = newSlice
			} else {
				totalRepeatedBlocks += 1
			}
		}

		if err != nil && err != io.EOF {
			if err == io.ErrUnexpectedEOF {
				break
			}

			logger.Error("erro ao ler o byte", "err", err)
			return ProcessResult{}, err
		}

		if err == io.EOF {
			break
		}
	}

	var finalHash [32]byte
	copy(finalHash[:], hasher.Sum(nil))

	return ProcessResult{
		storedHashes:        storedHashs,
		blockSequence:       blockSequence,
		totalBlock:          totalBlock,
		totalBytes:          totalBytes,
		totalRepeatedBlocks: totalRepeatedBlocks,
		originalFileHash:    finalHash,
	}, nil
}

func reconstructFile(storedHashes StoredHashes, blockSequence [][32]byte) ([]byte, error) {
	var reconstructed []byte
	for _, sequenceHash := range blockSequence {
		logger.Info("hashInSequence:", "info", sequenceHash)

		content, ok := storedHashes[sequenceHash]

		if !ok {
			return nil, errors.New("hash não existente")
		}

		if content == nil {
			return nil, errors.New("byte não existente")
		}

		convertedContent := helpers.ByteToHash(content)
		if convertedContent != sequenceHash {
			return nil, errors.New("integridade do bloco inválida")
		}

		reconstructed = append(reconstructed, content...)
	}

	return reconstructed, nil
}

func ReaderFile(path, storageDir, manifestDir string) {
	file, err := os.Open(path)
	if err != nil {
		logger.Error("Erro ao abrir o arquivo:", "err", err)
		return
	}

	defer file.Close()

	processResult, err := processByte(file)
	if err != nil {
		logger.Error("Erro ao processar os bytes do arquivo:", "err", err)
		return
	}

	logger.Info("Total de bytes lidos:", "info", processResult.totalBytes)
	logger.Info("Blocos lidos:", "info", processResult.totalBlock)
	logger.Info("Blocos únicos", "info", len(processResult.storedHashes))
	logger.Info("Blocos repetidos", "info", processResult.totalRepeatedBlocks)
	logger.Info("Sequência de Hashs", "info", processResult.blockSequence)
	logger.Info("Original", "info", processResult.originalFileHash)

	showStoredHashs(processResult.storedHashes)

	reconstructed, err := reconstructFile(processResult.storedHashes, processResult.blockSequence)
	if err != nil {
		logger.Error("Erro ao reconstruir o arquivo:", "err", err)
		return
	}

	reconstructedHash := helpers.ByteToHash(reconstructed)

	if reconstructedHash != processResult.originalFileHash {
		logger.Error("Hash calculado diverge do hash original")
		return
	}

	logger.Info("Original:", "info", processResult.originalFileHash)
	logger.Info("Reconstruído", "info", reconstructedHash)

	for hash, content := range processResult.storedHashes {
		if err := helpers.StoreBlock(storageDir, hash, content); err != nil {
			logger.Error("Erro ao armanzenar o hash:", "err", err)
			return
		}

		loadContent, err := helpers.LoadBlock(storageDir, hash)
		if err != nil {
			logger.Error("Erro ao carregar o hash:", "err", err)
			return
		}

		logger.Info("Conteúdo carregado:", "content", loadContent)
	}

	content, err := helpers.ReconstructFromDisk(storageDir, processResult.blockSequence)
	if err != nil {
		logger.Error("Erro ao recriar o arquivo do disco pela sequência de bytes", "err", err)
		return
	}

	if helpers.ByteToHash(content) != processResult.originalFileHash {
		logger.Error("Hash donteúdo recriado difere do hash calculado:", "recriado", helpers.ByteToHash(content), "esperado", processResult.originalFileHash)
		return
	}

	logger.Info("Conteúdo reconstruido", "info", content)

	reconstructedJSON, err := helpers.BuildManifestJSON(
		processResult.blockSequence,
		uint64(processResult.totalBytes),
		uint64(ChunkSize),
		path,
		helpers.HexToString(processResult.originalFileHash[:]),
	)
	if err != nil {
		logger.Error("Erro ao criar o json do manifesto", "err", err)
		return
	}

	logger.Info("JSON", "json", string(reconstructedJSON))

	saveManifestDir, err := helpers.SaveManifest(manifestDir, reconstructedJSON)
	if err != nil {
		logger.Error("Erro ao salvar o arquivo do manifesto:", "err", err)
		return
	}

	logger.Info("Caminho do manifesto:", "info", saveManifestDir)

	manifestData, err := helpers.LoadManifest(saveManifestDir)
	if err != nil {
		logger.Error("Erro ao recuperar o json do manifesto", "err", err)
		return
	}

	logger.Info("Dados do manifesto", "manifest", manifestData)

	decodeBlockHashesSequence, err := helpers.DecodeBlockSequence(manifestData.Blocks)
	if err != nil {
		logger.Error("Erro ao decodificar os hash do bloco de sequência do manifesto", "err", err)
		return
	}

	newReconstructed, err := helpers.ReconstructFromDisk(storageDir, decodeBlockHashesSequence)
	if err != nil {
		logger.Error("Erro ao recriar o arquivo do disco pela sequência de bytes recebida do manifesto", "err", err)
		return
	}

	reconstructedHash = helpers.ByteToHash(newReconstructed)

	if helpers.HexToString(reconstructedHash[:]) != manifestData.FileHash {
		logger.Error(
			"Hash reconstruído diverge do manifesto",
			"calculado", helpers.HexToString(reconstructedHash[:]),
			"esperado", manifestData.FileHash,
		)
		return
	}

	if uint64(len(newReconstructed)) != manifestData.FileSize {
		logger.Error("Tamanho reconstruído diverge do manifesto")
		return
	}

	logger.Info("Arquivos iguais")
}

func main() {
	wd, err := os.Getwd()
	if err != nil {
		logger.Error("Erro ao identificar o local de execução:", "err", err)
		os.Exit(1)
	}

	logger.Info("Local de execução:", "info", wd)

	storageDir, err := helpers.CreateInformedFolder(wd, helpers.StorageDir)
	if err != nil {
		logger.Error("Erro ao criar diretório de armazenamento:", "err", err)
		os.Exit(1)
	}

	manifestDir, err := helpers.CreateInformedFolder(wd, helpers.ManifestDir)
	if err != nil {
		logger.Error("Erro ao criar diretório de armazenamento do manifesto:", "err", err)
		os.Exit(1)
	}

	ReaderFile(filepath.Join(wd, "2026-10-20 15-42-30.mkv"), storageDir, manifestDir)

	/*
		restoredDir, err := helpers.CreateInformedFolder(wd, helpers.RestoredDir)
		if err != nil {
			logger.Error("Erro ao criar diretório de armazenamento de arquivos restaurados:", "err", err)
			os.Exit(1)
		}

		manifestPath := filepath.Join(
			manifestDir,
			"7186464ca1614a466f04ed3cb3d96ae20c7185be1679fa47c2fcd21c2621d4f6.json",
		)

		reconstructed, err := helpers.RestoreFromManifest(manifestPath, storageDir)
		if err != nil {
			logger.Error("Erro ao recupear o contéudo pelo manifesto:", "err", err)
			os.Exit(1)
		}

		logger.Info("Conteúdo reconstruido pelo manifesto:", "value", string(reconstructed))

		finalOutputPath := filepath.Join(restoredDir, "file.txt")
		if err := helpers.SaveRestoredFile(finalOutputPath, reconstructed); err != nil {
			logger.Error("Erro ao salvar o arquivo recupardo pelo contéudo do manifesto:", "err", err)
			os.Exit(1)
		}
	*/
}
