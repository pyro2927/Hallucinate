package main

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pyro2927/hallucinate/bandage_toss"
	"github.com/pyro2927/hallucinate/forge"
)

//const RIFT_HOST = "rift.mimic.lol"
const RIFT_HOST = "lvh.me"
const WEB_PROTOCOL = "http"
const WS_PROTOCOL = "ws"
const RIFT_HUB = WEB_PROTOCOL + "://" + RIFT_HOST
const HUB_TOKEN_FILE = "hub.token"

var secretKey []byte

// {"ok":true,"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoiNTMwMzA0IiwiaWF0IjoxNTYyMTkyNzc5fQ.4BXkkePT1W5dtwusPJD7Syo3lyfQBONVz2RA2Bgdf5M"}
type RiftResponse struct {
	Ok    bool   `json:"ok"`
	Token string `json:"token"`
}

type RiftPayload struct {
	PublicKey string `json:"pubkey"`
}

// {"code":"644763","iat":1562548258}
type JwtCode struct {
	Code string `json:"code"`
	Iat  int    `json:"iat"`
}

// {"secret":"MdTk081GkFQrSQ0pu4g/cY5gpcNbhUTjyR1uacAwnn8=","identity":"e7c69769-d606-4db7-b27a-236c2777f621","device":"iPhone","browser":"Safari"}
type DevicePayload struct {
	Secret   string `json:"secret"`
	Identity string `json:"identity"`
	Device   string `json:"device"`
	Browser  string `json:"browser"`
}

// https://github.com/molenzwiebel/Mimic/blob/master/conduit/HubConnectionHandler.cs#L109
type RiftOpCode int

const (
	Open    RiftOpCode = 1 // ex: [1,"aa8a1ecb-88bf-4ced-b778-92e4331fe453"]
	Message RiftOpCode = 2 // ex: [2,"6be5024a-81b1-4230-9c8c-3784960ec7e0","zmA3pYyKIrLYASZFIWuTdg==:Q2kz2vxHfM0gSOi3hmoh8w=="]
	Close   RiftOpCode = 3 // ex: [3,"cb8a1a22-0dd3-4e0e-a321-ebd3d904adc8"]
	Reply   RiftOpCode = 7
)

func getToken() string {
	valid := false
	data, err := ioutil.ReadFile(HUB_TOKEN_FILE)
	if err != nil {
		fmt.Println("No previous token found")
	} else {
		// ensure this token is still valid
		validateRequest, _ := http.Get(RIFT_HUB + "/check?token=" + string(data))
		result, _ := ioutil.ReadAll(validateRequest.Body)
		valid = (string(result) == "true")
		fmt.Println(string(data))
		fmt.Println(string(result))
		fmt.Println(valid)
	}
	if !valid {
		fmt.Println("Requesting one")
		pubKey := forge.PublicKey()
		body := &RiftPayload{PublicKey: pubKey}
		jsonValue, _ := json.Marshal(body)
		req, _ := http.Post(RIFT_HUB+"/register", "application/json", bytes.NewBuffer(jsonValue))
		response := RiftResponse{}
		body2, _ := ioutil.ReadAll(req.Body)
		json.Unmarshal(body2, &response)
		ioutil.WriteFile(HUB_TOKEN_FILE, []byte(response.Token), 0755)
		return response.Token
	}
	return string(data)
}

func accessCode() string {
	// TODO: do something about all these ignored errors
	data, _ := ioutil.ReadFile(HUB_TOKEN_FILE)
	chunk2 := strings.Split(string(data), ".")[1]
	// JWT doesn't pad base64 decoded strings, so we need to do that
	if i := len(chunk2) % 4; i != 0 {
		chunk2 += strings.Repeat("=", 4-i)
	}
	d, _ := base64.StdEncoding.DecodeString(chunk2)
	fmt.Println(string(d))
	j := JwtCode{}
	json.Unmarshal(d, &j)
	return j.Code
}

func sendMessage(ws *websocket.Conn, deviceId string, payload []interface{}) {
	// TODO: handle encrypting payload before sending
	var s []interface{}
	s = append(s, int(Reply))
	s = append(s, deviceId)
	s = append(s, payload)
	jsonData, _ := json.Marshal(s)

	// SEND IT!
	err := ws.WriteMessage(websocket.TextMessage, jsonData)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(string(jsonData))
}

func addBase64Padding(value string) string {
	m := len(value) % 4
	if m != 0 {
		value += strings.Repeat("=", 4-m)
	}
	return value
}

func Unpad(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])
	if unpadding > length {
		return nil, errors.New("unpad error. This could happen when incorrect encryption key is used")
	}
	return src[:(length - unpadding)], nil
}

func handleMessage(ws *websocket.Conn, deviceId string, payload interface{}) {
	switch payload := payload.(type) {
	case []interface{}:
		emsg := payload[1].(string)
		fmt.Println(emsg)
		bmsg, err := base64.StdEncoding.DecodeString(emsg)
		checkError(err)
		// Need to use the same settings as node-rsa
		// as that is what the client is using
		// https://github.com/rzcoder/node-rsa/blob/da04387da8897e70780a07aa9df88a198004a4b4/src/schemes/oaep.js#L25
		decoded, err := forge.PrivateKey().Decrypt(forge.Reader(), bmsg, &rsa.OAEPOptions{Hash: crypto.SHA1, Label: nil})
		if err != nil {
			fmt.Println("Unable to decrypt key")
			return
		}
		fmt.Println(string(decoded))
		var dp DevicePayload
		err = json.Unmarshal(decoded, &dp)
		checkError(err)
		fmt.Println(dp.Secret)
		secretKey, err = base64.StdEncoding.DecodeString(dp.Secret)
		checkError(err)
		// If all this works, confirm this is our device
		sendMessage(ws, deviceId, bandage_toss.Accept(true))
	case string:
		// <base 64 iv>:<base 64 encrypted data>
		ed := strings.Split(payload, ":")
		iv, err := base64.StdEncoding.DecodeString(addBase64Padding(ed[0]))
		checkError(err)
		fmt.Printf("AES Secret %s\n", base64.StdEncoding.EncodeToString(secretKey))
		fmt.Printf("IV %s\n", ed[0])
		fmt.Printf("Decrypting %s\n", ed[1])
		msg, err := base64.StdEncoding.DecodeString(addBase64Padding(ed[1]))
		checkError(err)
		if (len(msg) % aes.BlockSize) != 0 {
			fmt.Println("blocksize must be multipe of decoded message length")
			os.Exit(1)
		}
		// create our cipher block and decrypt
		block, err := aes.NewCipher(secretKey)
		checkError(err)
		cfb := cipher.NewCBCDecrypter(block, iv)
		cfb.CryptBlocks(msg, msg)
		unpadMsg, err := Unpad(msg)
		checkError(err)
		// Unmarshal, pass to decrypted message function
		var contents []interface{}
		err = json.Unmarshal(unpadMsg, &contents)
		checkError(err)
		handleDecryptedMessage(ws, deviceId, contents)
	}
}

func handleDecryptedMessage(ws *websocket.Conn, deviceId string, payload []interface{}) {
	fmt.Println(payload)
	switch bandage_toss.MobileOpCode(int(payload[0].(float64))) {
	case bandage_toss.Version:
		sendMessage(ws, deviceId, bandage_toss.VersionPayload())
	}
}

func main() {
	token := getToken()
	key := forge.PublicKey()
	u := url.URL{Scheme: WS_PROTOCOL, Host: RIFT_HOST, Path: "/conduit"}
	h := http.Header{"Token": {token}, "Public-Key": {key}}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	code := accessCode()
	fmt.Println("Token:", token)
	fmt.Println("Access Code:", code)
	// loop and listen to all messages
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("recv: %s", message)
		var contents []interface{}
		err = json.Unmarshal(message, &contents)
		if err != nil {
			log.Println("Error unmarshaling JSON:", err)
			continue
		}
		deviceId := contents[1].(string)
		switch RiftOpCode(int(contents[0].(float64))) {
		case Open:
			fmt.Println(deviceId + " connected")
		case Message:
			// get our key and iv
			handleMessage(c, deviceId, contents[2].(interface{}))
		case Close:
		case Reply:
		}
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
