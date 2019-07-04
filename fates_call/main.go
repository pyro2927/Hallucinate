package main

import (
	"crypto/tls"
	"encoding/base64"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

func main() {
	auth := base64.StdEncoding.EncodeToString([]byte("riot:" + password))

	u := url.URL{Scheme: "wss", Host: "127.0.0.1:" + port, Path: "/"}
	h := http.Header{"Authorization": {"Basic " + auth}}
	//connect to websocket server
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	dialer := websocket.Dialer{
		TLSClientConfig: config,
	}
	c, _, err := dialer.Dial(u.String(), h)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()
	// request updates for all events
	err = c.WriteMessage(websocket.TextMessage, []byte("[5,\"OnJsonApiEvent\"]"))
	if err != nil {
		log.Println("write:", err)
		return
	}
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("recv: %s", message)
	}
}
