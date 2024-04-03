package utils

import (
	"encoding/base32"
	"github.com/google/uuid"
)

func NewBrigadeID() string {
	bytes := [16]byte(uuid.New())
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes[:])
}
