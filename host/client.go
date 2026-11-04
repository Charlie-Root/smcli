package host

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Charlie-Root/smcli/config"
)

// Client talks Redfish to a single BMC. Use one Client per host; it is not shared between goroutines.
type Client struct {
	Host    config.Host
	Verbose bool
	DryRun  bool
	Log     io.Writer

	http        *http.Client
	user, pass  string
	systemPath  string
	managerPath string
	chassisPath string
}

// Result is what an operation returns: a human-readable text and structured data for JSON output.
type Result struct {
	Text string
	Data interface{}
}

func New(h config.Host, timeout time.Duration) (*Client, error) {
	user, pass, err := h.Credentials()
	if err != nil {
		return nil, err
	}
	if h.BMCAddress == "" {
		return nil, fmt.Errorf("host %s has no BMC address", h.Name)
	}
	tr := &http.Transport{
		// BMCs ship self-signed certificates.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &Client{
		Host: h,
		Log:  io.Discard,
		http: &http.Client{Transport: tr, Timeout: timeout},
		user: user,
		pass: pass,
	}, nil
}

// RedfishError is a non-2xx answer from the BMC.
type RedfishError struct {
	Method  string
	Path    string
	Status  int
	Message string
	Details []string
}

func (e *RedfishError) Error() string {
	msg := fmt.Sprintf("%s %s: HTTP %d", e.Method, e.Path, e.Status)
	if e.Message != "" {
		msg += ": " + e.Message
	}
	if len(e.Details) > 0 {
		msg += " (" + strings.Join(e.Details, "; ") + ")"
	}
	return msg
}

func parseError(method, path string, status int, body []byte) error {
	e := &RedfishError{Method: method, Path: path, Status: status}
	var r struct {
		Error struct {
			Code     string `json:"code"`
			Message  string `json:"message"`
			Extended []struct {
				Message    string `json:"Message"`
				Resolution string `json:"Resolution"`
			} `json:"@Message.ExtendedInfo"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &r) == nil && (r.Error.Message != "" || r.Error.Code != "") {
		e.Message = r.Error.Message
		if e.Message == "" {
			e.Message = r.Error.Code
		}
		for _, x := range r.Error.Extended {
			d := x.Message
			if x.Resolution != "" {
				d += " Resolution: " + x.Resolution
			}
			e.Details = append(e.Details, d)
		}
	} else if s := strings.TrimSpace(string(body)); s != "" && len(s) < 300 {
		e.Message = s
	}
	return e
}

func (c *Client) logf(format string, a ...interface{}) {
	if c.Verbose {
		fmt.Fprintf(c.Log, "[%s] "+format+"\n", append([]interface{}{c.Host.Name}, a...)...)
	}
}

func (c *Client) baseURL() string {
	addr := c.Host.BMCAddress
	if ip := net.ParseIP(addr); ip != nil && ip.To4() == nil {
		addr = "[" + addr + "]"
	}
	return "https://" + addr
}

func (c *Client) do(method, path string, payload interface{}) ([]byte, error) {
	var body []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = b
	}
	url := c.baseURL() + path
	c.logf("> %s %s %s", method, url, MaskSecrets(body))
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.user, c.pass)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("%s %s: reading response: %w", method, path, err)
	}
	c.logf("< %s (%d bytes)", resp.Status, len(respBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return respBody, parseError(method, path, resp.StatusCode, respBody)
	}
	return respBody, nil
}

func (c *Client) getJSON(path string, v interface{}) error {
	b, err := c.do("GET", path, nil)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("GET %s: decoding response: %w", path, err)
	}
	return nil
}

// write sends a changing request, or only describes it in dry-run mode.
func (c *Client) write(method, path string, payload interface{}, done string) (Result, error) {
	if c.DryRun {
		b, _ := json.Marshal(payload)
		return Result{
			Text: fmt.Sprintf("dry run: would %s %s %s", method, path, MaskSecrets(b)),
			Data: map[string]interface{}{"dry_run": true, "method": method, "path": path},
		}, nil
	}
	if _, err := c.do(method, path, payload); err != nil {
		return Result{}, err
	}
	return Result{Text: done, Data: map[string]interface{}{"method": method, "path": path}}, nil
}

// MaskSecrets hides any JSON field whose name contains "password".
func MaskSecrets(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	var m map[string]interface{}
	if json.Unmarshal(b, &m) != nil {
		return string(b)
	}
	for k := range m {
		if strings.Contains(strings.ToLower(k), "password") {
			m[k] = "****"
		}
	}
	out, _ := json.Marshal(m)
	return string(out)
}

func (c *Client) firstMember(collection string) (string, error) {
	var col struct {
		Members []struct {
			ID string `json:"@odata.id"`
		} `json:"Members"`
	}
	if err := c.getJSON(collection, &col); err != nil {
		return "", err
	}
	if len(col.Members) == 0 || col.Members[0].ID == "" {
		return "", fmt.Errorf("%s has no members", collection)
	}
	return col.Members[0].ID, nil
}

func (c *Client) system() (string, error) {
	if c.systemPath == "" {
		p, err := c.firstMember("/redfish/v1/Systems")
		if err != nil {
			return "", err
		}
		c.systemPath = p
	}
	return c.systemPath, nil
}

func (c *Client) manager() (string, error) {
	if c.managerPath == "" {
		p, err := c.firstMember("/redfish/v1/Managers")
		if err != nil {
			return "", err
		}
		c.managerPath = p
	}
	return c.managerPath, nil
}

func (c *Client) chassis() (string, error) {
	if c.chassisPath == "" {
		p, err := c.firstMember("/redfish/v1/Chassis")
		if err != nil {
			return "", err
		}
		c.chassisPath = p
	}
	return c.chassisPath, nil
}
