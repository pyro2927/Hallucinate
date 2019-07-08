package bandage_toss

import (
	"os"
	"strconv"
)

// https://github.com/molenzwiebel/Mimic/blob/master/conduit/MobileConnectionHandler.cs#L161
type MobileOpCode int

const (
	Secret          MobileOpCode = 1
	SecretResponse  MobileOpCode = 2
	Version         MobileOpCode = 3
	VersionResponse MobileOpCode = 4
	Subscribe       MobileOpCode = 5
	Unsubscribe     MobileOpCode = 6
	Request         MobileOpCode = 7
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
