package queue

import (
	"math/rand"
)

const randASCII = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(i int) string {
	b := make([]byte, i)
	for i := range b {
		b[i] = randASCII[rand.Intn(len(randASCII))]
	}
	return string(b)
}
