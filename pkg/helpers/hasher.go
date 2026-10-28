package helpers

import (
	"crypto/sha256"
	"hash"
)

func CreateHasherSHA256() hash.Hash {
	return sha256.New()
}
