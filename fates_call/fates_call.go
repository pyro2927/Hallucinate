package fates_call

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pyro2927/hallucinate/hawkshot"
)

type League struct {
	host string
	auth string
	ws   *websocket.Conn
	http *http.Client
}

type RequestCallback func(statusCode int, content interface{})

func (l *League) HandleRequest(path string, method string, body string, cb RequestCallback) {
	uri := fmt.Sprintf("https://%s%s", l.host, path)
	fmt.Printf("Making %s request to %s\n", method, uri)
	req, err := http.NewRequest(method, uri, strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
		return
	}
	req.Header.Add("Authorization", "Basic "+l.auth)
	if len(body) > 0 {
		req.Header.Add("Content-Type", "application/json")
	}
	// make request and parse response
	resp, err := l.http.Do(req)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer resp.Body.Close()
	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var content interface{}
	err = json.Unmarshal(r, &content)
	if err != nil {
		log.Fatal(err)
		return
	}
	cb(resp.StatusCode, content)
}

func LeagueConnection() *League {
	_, _, port, password, _ := hawkshot.LeagueCreds()
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
		os.Exit(1)
	}
	// request updates for all events
	err = c.WriteMessage(websocket.TextMessage, []byte("[5,\"OnJsonApiEvent\"]"))
	if err != nil {
		log.Println("write:", err)
		os.Exit(1)
	}
	var netClient = &http.Client{
		Timeout:   time.Second * 10,
		Transport: &http.Transport{TLSClientConfig: config},
	}
	return &League{ws: c, http: netClient, auth: auth, host: u.Host}
}
