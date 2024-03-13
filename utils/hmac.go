package utils

import "crypto/rand"

func GenHMACKey() ([]byte, error) {
	k := make([]byte, 32)
	_, err := rand.Read(k)
	if err != nil {
		return nil, err
	}
	return k, nil
}
