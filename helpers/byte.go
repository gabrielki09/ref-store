package helpers

import "crypto/sha256"

func ByteToHash(v []byte) [32]byte {
	return sha256.Sum256(v)
}
