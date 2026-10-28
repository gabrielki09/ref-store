package helpers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"ref-store/logger"
	"strings"
)

type Manifest struct {
	Version   int      `json:"version"`
	FileName  string   `json:"file_name"`
	FileSize  uint64   `json:"file_size"`
	ChunkSize uint64   `json:"chunk_size"`
	FileHash  string   `json:"file_hash"`
	Blocks    []string `json:"blocks"`
}

func BuildManifestJSON(
	blockSequence [][32]byte,
	fileSize,
	chunkSize uint64,
	path,
	fileHash string,
) ([]byte, error) {

	var manifest Manifest

	manifest.Version = 1
	manifest.FileName = filepath.Base(path)
	manifest.FileSize = fileSize
	manifest.ChunkSize = chunkSize
	manifest.FileHash = fileHash
	manifest.Blocks = make([]string, 0, len(blockSequence))

	for _, hashe := range blockSequence {
		manifest.Blocks = append(manifest.Blocks, HexToString(hashe[:]))
	}

	return json.MarshalIndent(manifest, "", "  ")
}

func SaveManifest(manifestDir string, jsonData []byte) (string, error) {

	jsonHashFileName := ByteToHash(jsonData)
	jsonHashFileNameString := fmt.Sprintf("%s.json", HexToString(jsonHashFileName[:]))

	manifestPath := filepath.Join(manifestDir, jsonHashFileNameString)

	if err := os.WriteFile(manifestPath, jsonData, 0666); err != nil {
		return "", err
	}

	return manifestPath, nil
}

func LoadManifest(manifestPath string) (Manifest, error) {
	baseFilePath := filepath.Base(manifestPath)
	hashFileName := strings.TrimSuffix(baseFilePath, ".json") // Pega o nome do arquivo sem .json

	var manifest Manifest

	manifestContent, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, err
	}

	calculatedHash := ByteToHash(manifestContent)
	calculatedHashHex := HexToString(calculatedHash[:])

	if calculatedHashHex != hashFileName {
		logger.Error(
			"Integridade do manifesto inválida",
			"calculado", calculatedHashHex,
			"esperado", hashFileName,
		)
		return Manifest{}, errors.New("integridade do manifesto inválida")
	}

	if err := json.Unmarshal(manifestContent, &manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func DecodeBlockSequence(blocks []string) ([][32]byte, error) {
	var decodedBlockSequence [][32]byte

	for _, hashInString := range blocks {
		var decodedHash [32]byte

		decodedBytes, err := hex.DecodeString(hashInString)

		if err != nil {
			logger.Error("Erro ao decodificar o hash em string:", err)
			return nil, err
		}

		if len(decodedBytes) != 32 {
			return nil, errors.New("hash deve conter exatamente 32 bytes")

		}

		copy(decodedHash[:], decodedBytes)
		decodedBlockSequence = append(decodedBlockSequence, decodedHash)
	}

	return decodedBlockSequence, nil
}

func RestoreFromManifest(manifestPath, storageDir string) ([]byte, string, error) {
	manifestData, err := LoadManifest(manifestPath)
	if err != nil {
		logger.Error("Erro ao carregar o manifesto:", "err", err)
		return nil, "", err
	}

	blocks, err := DecodeBlockSequence(manifestData.Blocks)
	if err != nil {
		logger.Error("Erro ao decodifica os blocos do manifesto:", err)
		return nil, "", err
	}

	reconstructed, err := ReconstructFromDisk(storageDir, blocks)
	if err != nil {
		logger.Error("Erro ao reconstruir os blocos do manifesto:", err)
		return nil, "", err
	}

	reconstructedHash := ByteToHash(reconstructed)
	reconstructedHashHex := HexToString(reconstructedHash[:])

	if reconstructedHashHex != manifestData.FileHash {
		logger.Error(
			"Hash reconstruído diverge do manifesto",
			"calculado", reconstructedHashHex,
			"esperado", manifestData.FileHash,
		)

		return nil, "", fmt.Errorf(
			"Hash reconstruído diverge do manifesto, calculado = %s, esperado = %s",
			reconstructedHashHex,
			manifestData.FileHash,
		)
	}

	if uint64(len(reconstructed)) != manifestData.FileSize {
		logger.Error("Tamanho reconstruído diverge do manifesto")

		return nil, "", errors.New("tamanho reconstruído diverge do manifesto")
	}

	return reconstructed, manifestData.FileName, nil
}

func ReconstructFromDisk(storageDir string, blockSequence [][32]byte) ([]byte, error) {
	var reconstructed []byte

	for _, hash := range blockSequence {
		content, err := LoadBlock(storageDir, hash)
		if err != nil {
			return nil, err
		}

		reconstructed = append(reconstructed, content...)
	}

	return reconstructed, nil
}
