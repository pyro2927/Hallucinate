package hawkshot

import (
	"io/ioutil"
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
				process, pid, port, password, protocol := readFile(install_dir)
				return process, pid, port, password, protocol
			}
		}
	}
}
