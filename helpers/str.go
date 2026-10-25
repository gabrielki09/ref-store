package helpers

import "encoding/hex"

func HexToString(s []byte) string {
	return hex.EncodeToString(s)
}

func DecodeStringToByte(s string) ([]byte, error) {
	return hex.DecodeString(s)
}
