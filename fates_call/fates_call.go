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
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pyro2927/hallucinate/braum"
	"github.com/pyro2927/hallucinate/hawkshot"
)

type League struct {
	host          string
	auth          string
	ws            *websocket.Conn
	http          *http.Client
	subscriptions []*regexp.Regexp
}

func (l *League) Subscribe(pattern string) error {
	subscription, err := regexp.Compile(pattern)
	// TODO: make this thread safe
	l.subscriptions = append(l.subscriptions, subscription)
	if err != nil {
		fmt.Printf("Unable to compile %s into Regex\n", pattern)
	}
	return err
}

type WebsocketEvent struct {
	Data      interface{} `json:"data"`
	EventType string      `json:"eventType"`
	Uri       string      `json:"uri"`
}

type WebsocketCallback func(content WebsocketEvent)

func (l *League) StartListening(wcb WebsocketCallback) {
	// Example message:
	// [8,"OnJsonApiEvent",{"data":[],"eventType":"Update","uri":"/lol-service-status/v1/ticker-messages"}]
	for {
		// TODO: handle server disconnect
		_, message, err := l.ws.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			continue
		}
		if len(message) < 8 {
			fmt.Println("Message too short, skipping Unmarshalling")
			continue
		}
		// hacky three-way marshal to get interface{} into struct
		var payload []interface{}
		err = braum.Unbreakable(message, &payload)
		if err != nil {
			log.Fatal(err)
			continue
		}
		b, err := json.Marshal(payload[2])
		if err != nil {
			log.Fatal(err)
			continue
		}
		var event WebsocketEvent
		err = braum.Unbreakable(b, &event)
		if err != nil {
			log.Fatal(err)
			continue
		}
		// verify that this URI has a regex match with one of our patterns
		for _, sub := range l.subscriptions {
			if sub.Match([]byte(event.Uri)) {
				wcb(event)
				break
			}
		}
	}
}

type RequestCallback func(statusCode int, content interface{})

func (l *League) HandleRequest(path string, method string, body string, cb RequestCallback) {
	uri := fmt.Sprintf("https://%s%s", l.host, path)
	//fmt.Printf("Making %s request to %s\n", method, uri)
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
	if len(r) > 0 {
		err = braum.Unbreakable(r, &content)
		if err != nil {
			log.Fatal(err)
			return
		}
	} else {
		content = []byte("null")
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
	return &League{ws: c, http: netClient, auth: auth, host: u.Host, subscriptions: []*regexp.Regexp{}}
}
