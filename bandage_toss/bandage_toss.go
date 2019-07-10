package bandage_toss

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pyro2927/hallucinate/fates_call"
)

// https://github.com/molenzwiebel/Mimic/blob/master/conduit/MobileConnectionHandler.cs#L161
type MobileOpCode int

const (
	Secret          MobileOpCode = 1
	SecretResponse  MobileOpCode = 2 // ex: [2, true]
	Version         MobileOpCode = 3 // ex: [3]
	VersionResponse MobileOpCode = 4 // ex: [4 "2.1.0" "localhost"]
	Subscribe       MobileOpCode = 5 // ex: [5 ^\/lol-lobby\/v2\/lobby$]
	Unsubscribe     MobileOpCode = 6
	Request         MobileOpCode = 7 // ex: [7 23 /lol-game-queues/v1/queues GET <nil>]
	RequestResponse MobileOpCode = 8
	Update          MobileOpCode = 9
)

func Accept(encrypted bool) []interface{} {
	var s []interface{}
	s = append(s, int(SecretResponse))
	s = append(s, strconv.FormatBool(encrypted))
	return s
}

func VersionPayload() []interface{} {
	var s []interface{}
	s = append(s, int(VersionResponse))
	s = append(s, "2.1.0") // stealing from https://github.com/molenzwiebel/Mimic/blob/master/conduit/Program.cs#L9
	hn, err := os.Hostname()
	if err != nil {
		s = append(s, "localhost")
	} else {
		s = append(s, hn)
	}
	return s
}

func RequestResponsePayload(requestId int, status int, content interface{}) []interface{} {
	var s []interface{}
	s = append(s, int(RequestResponse))
	s = append(s, requestId)
	s = append(s, status)
	s = append(s, content)
	return s
}

func UpdatePayload(event fates_call.WebsocketEvent) []interface{} {
	var s []interface{}
	s = append(s, int(Update))
	s = append(s, event.Uri)
	// https://github.com/molenzwiebel/Mimic/blob/master/conduit/MobileConnectionHandler.cs#L156
	if event.EventType == "Create" || event.EventType == "Update" {
		s = append(s, 200)
	} else {
		s = append(s, 200)
	}
	s = append(s, event.Data)
	fmt.Println("Update payload created")
	return s
}
