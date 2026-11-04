package cmd

import (
	"fmt"

	"github.com/Charlie-Root/smcli/config"
	"github.com/Charlie-Root/smcli/netbox"
)

type target struct {
	host config.Host
	err  error
}

func selecting() bool {
	return opts.rack != "" || opts.site != "" || opts.tag != "" || opts.role != ""
}

func resolveTargets(names []string) ([]target, error) {
	if len(names) > 0 && selecting() {
		return nil, fmt.Errorf("give host names or --rack/--site/--tag/--role, not both")
	}
	if len(names) == 0 && !selecting() {
		return nil, fmt.Errorf("no hosts given")
	}
	switch opts.source {
	case "file":
		if selecting() {
			return nil, fmt.Errorf("--rack/--site/--tag/--role need --source netbox")
		}
		hosts, err := config.LoadFile(opts.inventory)
		if err != nil {
			return nil, err
		}
		found, err := config.Find(hosts, names)
		if err != nil {
			return nil, err
		}
		out := make([]target, len(found))
		for i, h := range found {
			out[i] = target{host: h}
		}
		return out, nil
	case "netbox":
		nb, err := netbox.NewFromEnv(opts.timeout)
		if err != nil {
			return nil, err
		}
		res, err := nb.Resolve(netbox.Filter{Names: names, Rack: opts.rack, Site: opts.site, Tag: opts.tag, Role: opts.role})
		if err != nil {
			return nil, err
		}
		out := make([]target, len(res))
		for i, r := range res {
			out[i] = target{host: r.Host, err: r.Err}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unknown --source %q (want file or netbox)", opts.source)
	}
}
