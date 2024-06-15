package host

import (
	"log"

	"github.com/Charlie-Root/smcli/config"
)

var (
	CurrentHost config.Host
	RedfishPath string
	verbose     bool
)

func SetVerbose(v bool) {
	verbose = v
}

func SelectHost(name string) {
	for _, host := range config.ConfigData.Hosts {
		if host.Name == name {
			CurrentHost = host
			return
		}
	}
	log.Fatalf("Host with name %s not found", name)
}
