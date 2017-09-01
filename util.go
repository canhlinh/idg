package idg

import (
	"crypto/md5"
	"encoding/hex"
)

func HashMD5(data []byte) string {
	hasher := md5.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}
