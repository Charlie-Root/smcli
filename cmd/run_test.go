package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Charlie-Root/smcli/host"
)

func writeInventory(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "inventory.yaml")
	inv := "hosts:\n  - name: a\n    bmc_address: 192.0.2.1\n    username_password: ADMIN:x\n  - name: b\n    bmc_address: 192.0.2.2\n    username_password: ADMIN:y\n"
	if err := os.WriteFile(p, []byte(inv), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func resetOpts(inventory string) {
	opts = options{inventory: inventory, source: "file", parallel: 2, output: "text", timeout: 1}
}

func TestWriteOnManyHostsNeedsYes(t *testing.T) {
	resetOpts(writeInventory(t))
	called := false
	err := run([]string{"a", "b"}, true, func(c *host.Client) (host.Result, error) {
		called = true
		return host.Result{}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "--yes") || called {
		t.Fatalf("err = %v, called = %v", err, called)
	}
}

func TestSelectionFlagsNeedNetBox(t *testing.T) {
	resetOpts(writeInventory(t))
	opts.rack = "A.R10"
	if _, err := resolveTargets(nil); err == nil || !strings.Contains(err.Error(), "--source netbox") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnknownHost(t *testing.T) {
	resetOpts(writeInventory(t))
	if _, err := resolveTargets([]string{"a", "zzz"}); err == nil || !strings.Contains(err.Error(), "zzz") {
		t.Fatalf("err = %v", err)
	}
}

func TestFailedHostGivesErrFailed(t *testing.T) {
	resetOpts(writeInventory(t))
	var sb strings.Builder
	stdout = &sb
	defer func() { stdout = os.Stdout }()
	err := run([]string{"a"}, false, func(c *host.Client) (host.Result, error) {
		return host.Result{}, os.ErrDeadlineExceeded
	})
	if err != ErrFailed || !strings.Contains(sb.String(), "a: ERROR:") {
		t.Fatalf("err = %v, out = %q", err, sb.String())
	}
}
