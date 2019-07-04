package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pyro2927/hallucinate/forge"
)

const RIFT_HUB = "https://rift.mimic.lol"
const RIFT_WS = "wss://rift.mimic.lol/conduit"
const HUB_TOKEN_FILE = "hub.token"

// {"ok":true,"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoiNTMwMzA0IiwiaWF0IjoxNTYyMTkyNzc5fQ.4BXkkePT1W5dtwusPJD7Syo3lyfQBONVz2RA2Bgdf5M"}
type RiftResponse struct {
	Ok    bool   `json:"ok"`
	Token string `json:"token"`
}

type RiftPayload struct {
	PublicKey string `json:"pubkey"`
}

type JwtCode struct {
	Code string `json:"code"`
	Iat  int    `json:"iat"`
}

func getToken() string {
	data, err := ioutil.ReadFile(HUB_TOKEN_FILE)
	if err != nil {
		fmt.Println("No previous token found, requesting one")
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
	// TODO: verify token is still valid
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

func main() {
	token := getToken()
	key := forge.PublicKey()
	u := url.URL{Scheme: "wss", Host: "rift.mimic.lol", Path: "/conduit"}
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
	}
}
