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
	TotalRepeatedBlocks int
	TotalNewBlocks      int

	TotalNewBytes      uint64
	TotalReusedBytes   uint64
	TotalRepeatedBytes uint64

	TotalReusedBlocks int
	BlockSequence     [][32]byte
	OriginalFileHash  [32]byte
	TotalBlock        int
	TotalBytes        int
}

func ByteToHash(v []byte) [32]byte {
	return sha256.Sum256(v)
}

func ProcessByte(
	file *os.File,
	storageDir string,
) (ProcessResult, error) {
	b := make([]byte, ChunkSize)

	seenHashes := make(map[[32]byte]struct{})

	var blockSequence [][32]byte
	totalBlock := 0
	totalBytes := 0
	totalReusedBlocks := 0
	totalNewBlocks := 0

	var totalNewBytes uint64
	var totalReusedBytes uint64
	var totalRepeatedBytes uint64

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
				created, err := StoreBlock(storageDir, hash, b[:n])
				if err != nil {
					logger.Error("Erro ao salvar o bloco:", "err", err)
					return ProcessResult{}, err
				}

				if created {
					totalNewBlocks++
					totalNewBytes += uint64(n)
				} else {
					totalReusedBlocks++
					totalReusedBytes += uint64(n)
				}

				seenHashes[hash] = struct{}{}

			} else {
				totalRepeatedBlocks++
				totalRepeatedBytes += uint64(n)
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
		TotalNewBlocks:      totalNewBlocks,
		TotalReusedBlocks:   totalReusedBlocks,
		TotalNewBytes:       totalNewBytes,
		TotalReusedBytes:    totalReusedBytes,
		TotalRepeatedBytes:  totalRepeatedBytes,
	}, nil
}
