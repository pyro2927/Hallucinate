package forge

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/gob"
	"encoding/pem"
	"fmt"
	"io"
	"os"

	"github.com/pyro2927/pkcs8"
)

const CACHE_FILE = "rsa.key"

var instance io.Reader

func Reader() io.Reader {
	if instance == nil {
		instance = rand.Reader
	}
	return instance
}

func createOrLoadKey() *rsa.PrivateKey {
	dataFile, err := os.Open(CACHE_FILE)
	if err != nil {
		fmt.Println("No previous key file found, creating one")
		reader := Reader()
		bitSize := 2048

		key, err := rsa.GenerateKey(reader, bitSize)
		checkError(err)

		outFile, err := os.Create(CACHE_FILE)
		checkError(err)
		defer outFile.Close()

		encoder := gob.NewEncoder(outFile)
		err = encoder.Encode(key)
		checkError(err)
		return key
	}
	defer dataFile.Close()
	var key *rsa.PrivateKey
	dataDecoder := gob.NewDecoder(dataFile)
	err = dataDecoder.Decode(&key)
	checkError(err)
	return key
}

func PrivateKey() *rsa.PrivateKey {
	return createOrLoadKey()
}

func PublicKey() string {
	key := PrivateKey()
	asn1Bytes, err := pkcs8.ConvertPublicKeyToPKCS8(&key.PublicKey)
	checkError(err)
	var pemkey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}
	return string(pem.EncodeToMemory(pemkey))
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
