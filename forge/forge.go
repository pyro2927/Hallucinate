package forge

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/asn1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
)

const CACHE_FILE = "public.key"

func PublicKey() string {
	data, err := ioutil.ReadFile(CACHE_FILE)
	// if our program was unable to read the file
	// print out the reason why it can't
	if err != nil {
		fmt.Println("No previous key file found, creating one")
		reader := rand.Reader
		bitSize := 2048

		key, _ := rsa.GenerateKey(reader, bitSize)
		asn1Bytes, _ := asn1.Marshal(key.PublicKey)
		ioutil.WriteFile(CACHE_FILE, asn1Bytes, 0755)
		return base64.StdEncoding.EncodeToString(asn1Bytes)
	}
	return base64.StdEncoding.EncodeToString(data)
}
