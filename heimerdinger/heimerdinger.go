package heimerdinger

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

func addBase64Padding(value string) string {
	m := len(value) % 4
	if m != 0 {
		value += strings.Repeat("=", 4-m)
	}
	return value
}

func unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])
	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}
	return src[:(length - unpadding)], nil
}

func pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

// message is a string that consists of
// <Base64 encrypted IV>:<Base64 AES encrypted data>
func AESDecrypt(key []byte, message string) ([]interface{}, error) {
	ed := strings.Split(message, ":")
	iv, err := base64.StdEncoding.DecodeString(addBase64Padding(ed[0]))
	checkError(err)
	msg, err := base64.StdEncoding.DecodeString(addBase64Padding(ed[1]))
	checkError(err)
	if (len(msg) % aes.BlockSize) != 0 {
		fmt.Println("blocksize must be multipe of decoded message length")
		os.Exit(1)
	}
	// create our cipher block and decrypt
	block, err := aes.NewCipher(key)
	checkError(err)
	cfb := cipher.NewCBCDecrypter(block, iv)
	cfb.CryptBlocks(msg, msg)
	unpadMsg, err := unpad(msg)
	checkError(err)
	// Unmarshal, pass to decrypted message function
	var contents []interface{}
	err = json.Unmarshal(unpadMsg, &contents)
	checkError(err)
	return contents, nil
}

func AESEncrypt(key []byte, payload []interface{}) (string, error) {
	m, err := json.Marshal(payload)
	m = pad(m)
	checkError(err)
	// randomize our IV
	iv := make([]byte, 16)
	_, err = rand.Read(iv)
	checkError(err)
	// create cipher and decrypt
	block, err := aes.NewCipher(key)
	checkError(err)
	cfb := cipher.NewCBCEncrypter(block, iv)
	cfb.CryptBlocks(m, m)
	return strings.Join([]string{base64.StdEncoding.EncodeToString(iv), base64.StdEncoding.EncodeToString(m)}, ":"), nil
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
