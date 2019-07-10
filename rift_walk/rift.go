package main

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pyro2927/hallucinate/bandage_toss"
	"github.com/pyro2927/hallucinate/fates_call"
	"github.com/pyro2927/hallucinate/forge"
	"github.com/pyro2927/hallucinate/heimerdinger"
)

//const RIFT_HOST = "rift.mimic.lol"
const RIFT_HOST = "lvh.me"
const WEB_PROTOCOL = "http"
const WS_PROTOCOL = "ws"
const RIFT_HUB = WEB_PROTOCOL + "://" + RIFT_HOST
const HUB_TOKEN_FILE = "hub.token"

var secretKey []byte
var l *fates_call.League

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
	}
	if !valid {
		fmt.Println("Requesting one")
		pubKey := forge.PublicKey()
		body := &RiftPayload{PublicKey: pubKey}
		jsonValue, _ := json.Marshal(body)
		req, _ := http.Post(RIFT_HUB+"/register", "application/json", bytes.NewBuffer(jsonValue))
		response := RiftResponse{}
		body2, _ := ioutil.ReadAll(req.Body)
		braum.Unbreakable(body2, &response)
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
	j := JwtCode{}
	braum.Unbreakable(d, &j)
	return j.Code
}

func sendMessage(ws *websocket.Conn, deviceId string, payload interface{}) {
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
}

func sendSecureMessage(ws *websocket.Conn, deviceId string, payload []interface{}) {
	// AESEncrypt then send normally
	p, _ := heimerdinger.AESEncrypt(secretKey, payload)
	sendMessage(ws, deviceId, p)
}

func handleMessage(ws *websocket.Conn, deviceId string, payload interface{}) {
	switch payload := payload.(type) {
	case []interface{}:
		emsg := payload[1].(string)
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
		var dp DevicePayload
		err = braum.Unbreakable(decoded, &dp)
		checkError(err)
		secretKey, err = base64.StdEncoding.DecodeString(dp.Secret)
		checkError(err)
		// If all this works, confirm this is our device
		// TODO: make this an interactive check for yes/no
		sendMessage(ws, deviceId, bandage_toss.Accept(true))
		// subscribe to all future messages via websockets
		cb := func(event fates_call.WebsocketEvent) {
			sendSecureMessage(ws, deviceId, bandage_toss.UpdatePayload(event))
		}
		go l.StartListening(cb)
	case string:
		contents, _ := heimerdinger.AESDecrypt(secretKey, payload)
		handleDecryptedMessage(ws, deviceId, contents)
	}
}

func handleDecryptedMessage(ws *websocket.Conn, deviceId string, payload []interface{}) {
	switch bandage_toss.MobileOpCode(int(payload[0].(float64))) {
	case bandage_toss.Version:
		sendSecureMessage(ws, deviceId, bandage_toss.VersionPayload())
	case bandage_toss.Request:
		requestId := int(payload[1].(float64))
		path := payload[2].(string)
		method := payload[3].(string)
		body := ""
		// safe check for a payload vs nil
		switch payload[4].(type) {
		case string:
			body = payload[4].(string)
		}
		cb := func(status int, content interface{}) {
			sendSecureMessage(ws, deviceId, bandage_toss.RequestResponsePayload(requestId, status, content))
		}
		l.HandleRequest(path, method, body, cb)
	case bandage_toss.Subscribe:
		l.Subscribe(payload[1].(string))
	default:
		fmt.Println("Currently not handling payload...")
		fmt.Println(payload)
	}
}

func main() {
	// TODO: gracefully handle if LoL isn't open
	l = fates_call.LeagueConnection()
	fmt.Println("Connected to League!")
	fmt.Println("Connecting to Rift...")
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
	fmt.Println("Connected to Rift!")
	fmt.Println("Access Code:", code)
	// loop and listen to all messages
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		var contents []interface{}
		err = braum.Unbreakable(message, &contents)
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
