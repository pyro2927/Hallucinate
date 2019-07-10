package braum

import (
	"fmt"
	"encoding/json"
	"os"
	"runtime/debug"
)

func Unbreakable(b []byte, i interface{}) error{
	if len(b) <= 0 {
		fmt.Println("Attempting to unmarshal empty bytes, wtf!")
		debug.PrintStack()
		os.Exit(2)
	}
	return json.Unmarshal(b, i)
}
