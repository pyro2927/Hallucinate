package forge

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/gob"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

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

func UserHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func PreferencesDirectory() string {
	return UserHomeDir() + "/.mimic/"
}

func FileContents(filename string) []string {
	dat, err := ioutil.ReadFile(PreferencesDirectory() + filename)
	if err != nil {
		return []string{}
	}
	return strings.Split(string(dat), "\n")
}

func WriteLines(filename string, lines []string) error {
	os.MkdirAll(PreferencesDirectory(), os.ModePerm)
	return ioutil.WriteFile(PreferencesDirectory()+filename, []byte(strings.Join(lines, "\n")), 0644)
}
