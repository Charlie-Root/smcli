package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Host struct {
	Name             string `yaml:"name" json:"name"`
	BMCAddress       string `yaml:"bmc_address" json:"bmc_address"`
	UsernamePassword string `yaml:"username_password" json:"-"`
	ISOImage         string `yaml:"iso_image" json:"iso_image,omitempty"`
}

type Config struct {
	Hosts []Host `yaml:"hosts"`
}

// Credentials splits the "user:password" field.
func (h Host) Credentials() (string, string, error) {
	parts := strings.SplitN(h.UsernamePassword, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", fmt.Errorf("host %s: username_password must be \"user:password\"", h.Name)
	}
	return parts[0], parts[1], nil
}

// LoadFile reads an inventory file; "-" reads it from stdin.
func LoadFile(path string) ([]Host, error) {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening inventory: %w", err)
		}
		defer f.Close()
		r = f
	}
	var c Config
	if err := yaml.NewDecoder(r).Decode(&c); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("decoding inventory %s: %w", path, err)
	}
	return c.Hosts, nil
}

// Find returns the named hosts, in the requested order.
func Find(hosts []Host, names []string) ([]Host, error) {
	idx := make(map[string]Host, len(hosts))
	for _, h := range hosts {
		idx[h.Name] = h
	}
	out := make([]Host, 0, len(names))
	var missing []string
	for _, n := range names {
		h, ok := idx[n]
		if !ok {
			missing = append(missing, n)
			continue
		}
		out = append(out, h)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("host(s) not found in inventory: %s", strings.Join(missing, ", "))
	}
	return out, nil
}
