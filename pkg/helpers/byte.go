package helpers

import (
	"crypto/sha256"
	"io"
	"os"
	"ref-store/logger"
)

const ChunkSize = 64 * 1024

type ProcessResult struct {
	TotalUniqueBlocks   int
	BlockSequence       [][32]byte
	OriginalFileHash    [32]byte
	TotalBlock          int
	TotalBytes          int
	TotalRepeatedBlocks int
}

func ByteToHash(v []byte) [32]byte {
	return sha256.Sum256(v)
}

func ProcessByte(file *os.File, storageDir string) (ProcessResult, error) {
	b := make([]byte, ChunkSize)

	seenHashes := make(map[[32]byte]struct{})

	var blockSequence [][32]byte
	totalBlock := 0
	totalBytes := 0
	totalRepeatedBlocks := 0

	hasher := CreateHasherSHA256()

	for {
		n, err := io.ReadFull(file, b)

		if n > 0 {
			totalBlock += 1
			totalBytes += n

			hasher.Write(b[:n])
			hash := ByteToHash(b[:n])

			logger.Debug("Hash gerado: ", "hash", hash)

			blockSequence = append(blockSequence, hash)

			if _, ok := seenHashes[hash]; !ok {
				if err := StoreBlock(storageDir, hash, b[:n]); err != nil {
					logger.Error("Erro ao salvar o bloco:", "err", err)
					return ProcessResult{}, err
				}

				if _, err := LoadBlock(storageDir, hash); err != nil {
					return ProcessResult{}, err
				}

				seenHashes[hash] = struct{}{}

			} else {
				totalRepeatedBlocks++
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
		TotalUniqueBlocks:   len(seenHashes),
		BlockSequence:       blockSequence,
		TotalBlock:          totalBlock,
		TotalBytes:          totalBytes,
		TotalRepeatedBlocks: totalRepeatedBlocks,
		OriginalFileHash:    finalHash,
	}, nil
}
