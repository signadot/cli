package utils

import (
	"math/rand"
	"time"
)

func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	b := make([]rune, n)

	for i := 0; i < n; i++ {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}
