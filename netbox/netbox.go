// Package netbox resolves BMC address and credentials from NetBox:
// device -> oob_ip -> assigned interface -> custom fields ipmi_username / ipmi_password.
package netbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Charlie-Root/smcli/config"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// NewFromEnv uses NETBOX_URL and NETBOX_TOKEN.
func NewFromEnv(timeout time.Duration) (*Client, error) {
	u := strings.TrimRight(os.Getenv("NETBOX_URL"), "/")
	t := os.Getenv("NETBOX_TOKEN")
	if u == "" || t == "" {
		return nil, fmt.Errorf("NETBOX_URL and NETBOX_TOKEN must be set for --source netbox")
	}
	return New(u, t, timeout), nil
}

func New(baseURL, token string, timeout time.Duration) *Client {
	// No keep-alive: long-lived connections through the NetBox load balancer can stall.
	tr := &http.Transport{Proxy: http.ProxyFromEnvironment, DisableKeepAlives: true}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, HTTP: &http.Client{Transport: tr, Timeout: timeout}}
}

// Filter selects devices either by name or by rack/site/tag/role.
type Filter struct {
	Names []string
	Rack  string
	Site  string
	Tag   string
	Role  string
}

// Target is one selected device; Err is set when its BMC cannot be resolved.
type Target struct {
	Host config.Host
	Err  error
}

type device struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	OOBIP *struct {
		ID int `json:"id"`
	} `json:"oob_ip"`
}

type page struct {
	Next    string          `json:"next"`
	Results json.RawMessage `json:"results"`
}

func (c *Client) get(path string, q url.Values, v interface{}) error {
	u := path
	if !strings.HasPrefix(path, "http") {
		u = c.BaseURL + path
		if len(q) > 0 {
			u += "?" + q.Encode()
		}
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+c.Token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("netbox: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return fmt.Errorf("netbox: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return fmt.Errorf("netbox: GET %s: HTTP %d %s", path, resp.StatusCode, msg)
	}
	return json.Unmarshal(body, v)
}

func (c *Client) devices(q url.Values) ([]device, error) {
	q.Set("limit", "100")
	q.Set("fields", "id,name,oob_ip")
	var out []device
	next := "/api/dcim/devices/"
	for next != "" {
		var p page
		if err := c.get(next, q, &p); err != nil {
			return nil, err
		}
		var ds []device
		if err := json.Unmarshal(p.Results, &ds); err != nil {
			return nil, err
		}
		out = append(out, ds...)
		next, q = p.Next, nil
	}
	return out, nil
}

// Resolve returns one Target per selected device, in NetBox order (or the order of Names).
func (c *Client) Resolve(f Filter) ([]Target, error) {
	var devs []device
	if len(f.Names) > 0 {
		for _, n := range f.Names {
			ds, err := c.devices(url.Values{"name": {n}})
			if err != nil {
				return nil, err
			}
			if len(ds) == 0 {
				devs = append(devs, device{Name: n, ID: -1})
				continue
			}
			devs = append(devs, ds[0])
		}
	} else {
		q := url.Values{}
		if f.Site != "" {
			q.Set("site", f.Site)
		}
		if f.Tag != "" {
			q.Set("tag", f.Tag)
		}
		if f.Role != "" {
			q.Set("role", f.Role)
		}
		if f.Rack != "" {
			rq := url.Values{"name": {f.Rack}, "fields": {"id"}}
			if f.Site != "" {
				rq.Set("site", f.Site)
			}
			var p page
			if err := c.get("/api/dcim/racks/", rq, &p); err != nil {
				return nil, err
			}
			var racks []struct {
				ID int `json:"id"`
			}
			if err := json.Unmarshal(p.Results, &racks); err != nil {
				return nil, err
			}
			if len(racks) == 0 {
				return nil, fmt.Errorf("netbox: rack %q not found", f.Rack)
			}
			for _, r := range racks {
				q.Add("rack_id", fmt.Sprint(r.ID))
			}
		}
		if len(q) == 0 {
			return nil, fmt.Errorf("netbox: no selection given")
		}
		var err error
		if devs, err = c.devices(q); err != nil {
			return nil, err
		}
	}

	out := make([]Target, 0, len(devs))
	for _, d := range devs {
		t := Target{Host: config.Host{Name: d.Name}}
		if d.ID < 0 {
			t.Err = fmt.Errorf("device not found in NetBox")
		} else {
			t.Host, t.Err = c.bmc(d)
		}
		out = append(out, t)
	}
	return out, nil
}

func (c *Client) bmc(d device) (config.Host, error) {
	h := config.Host{Name: d.Name}
	if d.OOBIP == nil {
		return h, fmt.Errorf("no OOB IP set in NetBox")
	}
	var ip struct {
		Address            string `json:"address"`
		AssignedObjectType string `json:"assigned_object_type"`
		AssignedObjectID   *int   `json:"assigned_object_id"`
	}
	if err := c.get(fmt.Sprintf("/api/ipam/ip-addresses/%d/", d.OOBIP.ID), url.Values{"fields": {"address,assigned_object_type,assigned_object_id"}}, &ip); err != nil {
		return h, err
	}
	h.BMCAddress = strings.SplitN(ip.Address, "/", 2)[0]
	if ip.AssignedObjectType != "dcim.interface" || ip.AssignedObjectID == nil {
		return h, fmt.Errorf("OOB IP %s is not assigned to an interface", ip.Address)
	}
	var iface struct {
		Name         string                 `json:"name"`
		CustomFields map[string]interface{} `json:"custom_fields"`
	}
	if err := c.get(fmt.Sprintf("/api/dcim/interfaces/%d/", *ip.AssignedObjectID), url.Values{"fields": {"name,custom_fields"}}, &iface); err != nil {
		return h, err
	}
	user, _ := iface.CustomFields["ipmi_username"].(string)
	pass, _ := iface.CustomFields["ipmi_password"].(string)
	if user == "" || pass == "" {
		return h, fmt.Errorf("no IPMI credentials on interface %s", iface.Name)
	}
	h.UsernamePassword = user + ":" + pass
	return h, nil
}
