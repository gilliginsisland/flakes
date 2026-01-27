package menuet

import (
	"math/rand"
	"time"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIndexBits = 6
	letterIndexMask = 1<<letterIndexBits - 1
	letterIndexMax  = 63 / letterIndexBits
)

var randomSource = rand.NewSource(time.Now().UnixNano())

// RandomString returns a random string of the given length
func randomString(length int) string {
	b := make([]byte, length)
	for i, cache, remain := length-1, randomSource.Int63(), letterIndexMax; i >= 0; {
		if remain == 0 {
			cache, remain = randomSource.Int63(), letterIndexMax
		}
		if index := int(cache & letterIndexMask); index < len(letterBytes) {
			b[i] = letterBytes[index]
			i--
		}
		cache >>= letterIndexBits
		remain--
	}
	return string(b)
}

// ArbitraryKeyNotInMap returns an arbitrary string key that is not yet used in the map
func randomKeyNotInMap[V any](m map[string]V) string {
	length := 3 + len(m)/len(letterBytes)
	for count := 0; ; count++ {
		key := randomString(length)
		if _, exists := m[key]; !exists {
			return key
		}
		if count > length*len(letterBytes) {
			// This map must be quite full
			length++
		}
	}
}
