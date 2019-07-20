package braum

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/pyro2927/hallucinate/forge"
)

// {"secret":"MdTk081GkFQrSQ0pu4g/cY5gpcNbhUTjyR1uacAwnn8=","identity":"e7c69769-d606-4db7-b27a-236c2777f621","device":"iPhone","browser":"Safari"}
type DevicePayload struct {
	Secret   string `json:"secret"`
	Identity string `json:"identity"`
	Device   string `json:"device"`
	Browser  string `json:"browser"`
}

func Unbreakable(b []byte, i interface{}) error {
	if len(b) <= 0 {
		fmt.Println("Attempting to unmarshal empty bytes, wtf!")
		debug.PrintStack()
		os.Exit(2)
	}
	return json.Unmarshal(b, i)
}

func DeviceApproved(dp DevicePayload) bool {
	// check if previously approved
	approvedDevices := forge.FileContents("devices.txt")
	for _, d := range approvedDevices {
		if dp.Identity == d {
			return true
		}
	}
	// otherwise prompt
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s (%s) is trying to connect, allow? [y/N]", dp.Device, dp.Browser)
	text, _ := reader.ReadString('\n')
	approved := strings.HasPrefix(strings.TrimLeft(text, "	  "), "y")
	if approved {
		forge.WriteLines("devices.txt", append(approvedDevices, dp.Identity))
	}
	return approved
}
