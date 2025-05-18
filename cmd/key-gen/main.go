package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
)

const DefaultTokenLength = 32

func main() {
	token := fmt.Sprintf("go_idasen_desk_%s", GenerateSecureTokenHex(DefaultTokenLength))

	fmt.Fprintf(os.Stdout, "Generated token: %s\n", token)
}

func GenerateSecureTokenHex(length int) string {
	// Create a byte slice to store random bytes
	b := make([]byte, length)

	// Fill the slice with random bytes
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	// Encode the random bytes as a hex string
	return hex.EncodeToString(b)
}
