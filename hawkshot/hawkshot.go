package hawkshot

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/shirou/gopsutil/process"
)

func readFile(location string) (string, string, string, string, string) {
	dat, _ := ioutil.ReadFile(location + "/lockfile")
	x := strings.Split(string(dat), ":")
	return x[0], x[1], x[2], x[3], x[4]
}

// returns information for League client
func LeagueCreds() (string, string, string, string, string) {
	// TODO: this is slow as fuck, should figure out a way to do this faster
	fmt.Println("Hunting for LeagueClientUX process... (slow)")
	processes, _ := process.Processes()
	r, _ := regexp.Compile("--install-directory=(.*?) --")

	// loop over all processes
	for _, proc := range processes {
		e, _ := proc.Exe()
		// find league client one
		if strings.Contains(e, "LeagueClientUx") {
			cli, _ := proc.Cmdline()
			if r.MatchString(cli) {
				install_dir := r.FindStringSubmatch(cli)[1]
				proc, pid, port, password, protocol := readFile(install_dir)
				return proc, pid, port, password, protocol
			}
		}
	}
	// TODO: better handle League not running
	fmt.Println("League not running, try again")
	os.Exit(1)
	return "", "", "", "", ""
}
