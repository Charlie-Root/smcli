package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/Charlie-Root/smcli/host"
)

type opFunc func(c *host.Client) (host.Result, error)

type outcome struct {
	Host    string      `json:"host"`
	BMC     string      `json:"bmc_address,omitempty"`
	OK      bool        `json:"ok"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

var stdout io.Writer = os.Stdout

// run resolves the hosts and applies op to each, in parallel. write marks operations that change a host.
func run(names []string, write bool, op opFunc) error {
	if opts.output != "text" && opts.output != "json" {
		return fmt.Errorf("unknown --output %q (want text or json)", opts.output)
	}
	targets, err := resolveTargets(names)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no hosts selected")
	}
	if write && !opts.dryRun && len(targets) > 1 && !opts.yes {
		return fmt.Errorf("refusing to change %d hosts without --yes (use --dry-run to preview)", len(targets))
	}

	results := make([]outcome, len(targets))
	sem := make(chan struct{}, max(1, opts.parallel))
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t target) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = execute(t, op)
		}(i, t)
	}
	wg.Wait()

	failed := 0
	for _, r := range results {
		if !r.OK {
			failed++
		}
	}
	if opts.output == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return err
		}
	} else {
		printText(results)
	}
	if len(results) > 1 {
		fmt.Fprintf(os.Stderr, "%d ok, %d failed\n", len(results)-failed, failed)
	}
	if failed > 0 {
		return ErrFailed
	}
	return nil
}

func execute(t target, op opFunc) outcome {
	o := outcome{Host: t.host.Name, BMC: t.host.BMCAddress}
	if t.err != nil {
		o.Error = t.err.Error()
		return o
	}
	c, err := host.New(t.host, opts.timeout)
	if err != nil {
		o.Error = err.Error()
		return o
	}
	c.Verbose, c.DryRun, c.Log = opts.verbose, opts.dryRun, os.Stderr
	res, err := op(c)
	if err != nil {
		o.Error = err.Error()
		return o
	}
	o.OK, o.Message, o.Data = true, res.Text, res.Data
	return o
}

func printText(results []outcome) {
	for _, r := range results {
		switch {
		case !r.OK:
			fmt.Fprintf(stdout, "%s: ERROR: %s\n", r.Host, r.Error)
		case strings.Contains(r.Message, "\n"):
			fmt.Fprintf(stdout, "%s:\n  %s\n", r.Host, strings.ReplaceAll(r.Message, "\n", "\n  "))
		default:
			fmt.Fprintf(stdout, "%s: %s\n", r.Host, r.Message)
		}
	}
}
