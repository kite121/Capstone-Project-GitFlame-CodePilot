package repository

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		encoded := hex.EncodeToString(b[:])
		return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
	}
	return strconv.FormatInt(time.Now().UnixNano(), 16)
}
