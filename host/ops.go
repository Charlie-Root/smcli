package host

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

var resetTypes = map[string]string{
	"on":       "On",
	"off":      "ForceOff",
	"restart":  "ForceRestart",
	"shutdown": "GracefulShutdown",
}

// Reset changes the power state: on, off (forced), restart (forced) or shutdown (graceful).
func (c *Client) Reset(action string) (Result, error) {
	t, ok := resetTypes[action]
	if !ok {
		return Result{}, fmt.Errorf("unknown power action %q", action)
	}
	sys, err := c.system()
	if err != nil {
		return Result{}, err
	}
	return c.write("POST", sys+"/Actions/ComputerSystem.Reset", map[string]string{"ResetType": t}, "power "+t+" accepted")
}

func (c *Client) PowerStatus() (Result, error) {
	sys, err := c.system()
	if err != nil {
		return Result{}, err
	}
	var s struct{ PowerState string }
	if err := c.getJSON(sys, &s); err != nil {
		return Result{}, err
	}
	return Result{Text: "power " + s.PowerState, Data: map[string]string{"power_state": s.PowerState}}, nil
}

var bootTargets = map[string]string{"cd": "Cd", "pxe": "Pxe"}

// BootOnce sets a one-time UEFI boot override.
func (c *Client) BootOnce(target string) (Result, error) {
	t, ok := bootTargets[target]
	if !ok {
		return Result{}, fmt.Errorf("unknown boot target %q", target)
	}
	sys, err := c.system()
	if err != nil {
		return Result{}, err
	}
	payload := map[string]interface{}{"Boot": map[string]string{
		"BootSourceOverrideEnabled": "Once",
		"BootSourceOverrideTarget":  t,
		"BootSourceOverrideMode":    "UEFI",
	}}
	return c.write("PATCH", sys, payload, "next boot "+t+" (once, UEFI)")
}

const virtualMediaCD = "/redfish/v1/Managers/1/VirtualMedia/CD1"

func (c *Client) MediaInsert(image string) (Result, error) {
	if image == "" {
		return Result{}, fmt.Errorf("no ISO image: set iso_image in the inventory or pass --iso")
	}
	return c.write("POST", virtualMediaCD+"/Actions/VirtualMedia.InsertMedia", map[string]string{"Image": image}, "inserted "+image)
}

func (c *Client) MediaEject() (Result, error) {
	return c.write("POST", virtualMediaCD+"/Actions/VirtualMedia.EjectMedia", map[string]string{}, "ejected")
}

func (c *Client) MediaStatus() (Result, error) {
	var m map[string]interface{}
	err := c.getJSON("/redfish/v1/Managers/1/VirtualMedia/1", &m)
	if err != nil {
		if err2 := c.getJSON(virtualMediaCD, &m); err2 != nil {
			return Result{}, err
		}
	}
	return Result{Text: fmt.Sprintf("inserted=%v image=%v", m["Inserted"], m["Image"]), Data: m}, nil
}

// CreateUser adds an Administrator account.
func (c *Client) CreateUser(username, password string) (Result, error) {
	payload := map[string]interface{}{"UserName": username, "Password": password, "RoleId": "Administrator", "Enabled": true}
	return c.write("POST", "/redfish/v1/AccountService/Accounts", payload, "user "+username+" created")
}

const ipAccessControl = "/redfish/v1/Managers/1/Oem/Supermicro/IPAccessControl"

func (c *Client) EnableACL() (Result, error) {
	return c.write("PATCH", ipAccessControl, map[string]bool{"ServiceEnabled": true}, "IP access control enabled")
}

func (c *Client) AddACL(address string, prefix int, policy string) (Result, error) {
	payload := map[string]interface{}{"Address": address, "PrefixLength": prefix, "Policy": policy}
	return c.write("POST", ipAccessControl+"/FilterRules", payload, fmt.Sprintf("rule %s/%d %s added", address, prefix, policy))
}

type status struct {
	Health       string `json:"Health"`
	HealthRollup string `json:"HealthRollup"`
	State        string `json:"State"`
}

// Info summarises the system and its BMC.
func (c *Client) Info() (Result, error) {
	sys, err := c.system()
	if err != nil {
		return Result{}, err
	}
	var s struct {
		Manufacturer     string
		Model            string
		SerialNumber     string
		BiosVersion      string
		PowerState       string
		Status           status
		ProcessorSummary struct {
			Count int
			Model string
		}
		MemorySummary struct {
			TotalSystemMemoryGiB float64
		}
	}
	if err := c.getJSON(sys, &s); err != nil {
		return Result{}, err
	}
	data := map[string]interface{}{
		"manufacturer":  s.Manufacturer,
		"model":         s.Model,
		"serial":        s.SerialNumber,
		"bios_version":  s.BiosVersion,
		"power_state":   s.PowerState,
		"health":        s.Status.Health,
		"health_rollup": s.Status.HealthRollup,
		"cpu_model":     s.ProcessorSummary.Model,
		"cpu_count":     s.ProcessorSummary.Count,
		"memory_gib":    s.MemorySummary.TotalSystemMemoryGiB,
	}
	bmc := "unknown"
	if mgr, err := c.manager(); err == nil {
		var m struct{ FirmwareVersion, Model string }
		if c.getJSON(mgr, &m) == nil {
			bmc = m.FirmwareVersion
			data["bmc_firmware"] = m.FirmwareVersion
			data["bmc_model"] = m.Model
		}
	}
	text := fmt.Sprintf("%s %s, serial %s\npower %s, health %s (rollup %s)\nBIOS %s, BMC firmware %s\nCPU %dx %s, memory %.0f GiB",
		s.Manufacturer, s.Model, s.SerialNumber, s.PowerState, orDash(s.Status.Health), orDash(s.Status.HealthRollup),
		s.BiosVersion, bmc, s.ProcessorSummary.Count, s.ProcessorSummary.Model, s.MemorySummary.TotalSystemMemoryGiB)
	return Result{Text: text, Data: data}, nil
}

// Sensor is one temperature, fan, PSU or voltage reading.
type Sensor struct {
	Kind     string   `json:"kind"`
	Name     string   `json:"name"`
	Reading  *float64 `json:"reading,omitempty"`
	Unit     string   `json:"unit,omitempty"`
	Critical *float64 `json:"critical,omitempty"`
	Health   string   `json:"health,omitempty"`
	State    string   `json:"state,omitempty"`
}

func (s Sensor) ok() bool { return s.Health == "" || s.Health == "OK" }

// Sensors reads the chassis Thermal and Power resources.
func (c *Client) Sensors(problemsOnly bool) (Result, error) {
	ch, err := c.chassis()
	if err != nil {
		return Result{}, err
	}
	var sensors []Sensor
	var errs []string

	var th struct {
		Temperatures []struct {
			Name                   string
			ReadingCelsius         *float64
			UpperThresholdCritical *float64
			Status                 status
		}
		Fans []struct {
			Name         string
			FanName      string
			Reading      *float64
			ReadingUnits string
			Status       status
		}
	}
	if err := c.getJSON(ch+"/Thermal", &th); err != nil {
		errs = append(errs, err.Error())
	}
	for _, t := range th.Temperatures {
		sensors = append(sensors, Sensor{"temperature", t.Name, t.ReadingCelsius, "C", t.UpperThresholdCritical, t.Status.Health, t.Status.State})
	}
	for _, f := range th.Fans {
		name := f.Name
		if name == "" {
			name = f.FanName
		}
		sensors = append(sensors, Sensor{"fan", name, f.Reading, f.ReadingUnits, nil, f.Status.Health, f.Status.State})
	}

	var pw struct {
		PowerSupplies []struct {
			Name             string
			PowerInputWatts  *float64
			LineInputVoltage *float64
			Status           status
		}
		Voltages []struct {
			Name                   string
			ReadingVolts           *float64
			UpperThresholdCritical *float64
			Status                 status
		}
	}
	if err := c.getJSON(ch+"/Power", &pw); err != nil {
		errs = append(errs, err.Error())
	}
	for _, p := range pw.PowerSupplies {
		sensors = append(sensors, Sensor{"psu", p.Name, p.PowerInputWatts, "W", nil, p.Status.Health, p.Status.State})
	}
	for _, v := range pw.Voltages {
		sensors = append(sensors, Sensor{"voltage", v.Name, v.ReadingVolts, "V", v.UpperThresholdCritical, v.Status.Health, v.Status.State})
	}
	if len(errs) == 2 {
		return Result{}, fmt.Errorf("%s", strings.Join(errs, "; "))
	}

	var shown []Sensor
	bad := 0
	for _, s := range sensors {
		if s.State == "Absent" {
			continue
		}
		if !s.ok() {
			bad++
		}
		if problemsOnly && s.ok() {
			continue
		}
		shown = append(shown, s)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d sensors, %d not OK", len(sensors), bad)
	for _, s := range shown {
		mark := " "
		if !s.ok() {
			mark = "!"
		}
		reading := "n/a"
		if s.Reading != nil {
			reading = fmt.Sprintf("%.1f %s", *s.Reading, s.Unit)
		}
		line := fmt.Sprintf("\n%s %-11s %-28s %s", mark, s.Kind, s.Name, reading)
		if s.Critical != nil {
			line += fmt.Sprintf(" (crit %.1f)", *s.Critical)
		}
		if !s.ok() {
			line += " " + s.Health
		}
		b.WriteString(line)
	}
	for _, e := range errs {
		b.WriteString("\nwarning: " + e)
	}
	return Result{Text: b.String(), Data: map[string]interface{}{"not_ok": bad, "sensors": shown, "warnings": errs}}, nil
}

// LogEntry is one hardware/event log entry.
type LogEntry struct {
	Log      string `json:"log"`
	ID       string `json:"id"`
	Created  string `json:"created"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	created  time.Time
}

const maxLogEntries = 5000

// SEL reads the system event logs (and optionally the BMC's own logs), newest first.
func (c *Client) SEL(since time.Duration, limit int, includeManager bool) (Result, error) {
	sys, err := c.system()
	if err != nil {
		return Result{}, err
	}
	collections := []string{sys + "/LogServices"}
	if includeManager {
		if mgr, err := c.manager(); err == nil {
			collections = append(collections, mgr+"/LogServices")
		}
	}
	var services []string
	for _, col := range collections {
		var l struct {
			Members []struct {
				ID string `json:"@odata.id"`
			} `json:"Members"`
		}
		if err := c.getJSON(col, &l); err != nil {
			return Result{}, err
		}
		for _, m := range l.Members {
			services = append(services, m.ID)
		}
	}
	if len(services) == 0 {
		return Result{}, fmt.Errorf("no log services found")
	}

	cutoff := time.Time{}
	if since > 0 {
		cutoff = time.Now().Add(-since)
	}
	var entries []LogEntry
	for _, svc := range services {
		next := svc + "/Entries"
		for next != "" && len(entries) < maxLogEntries {
			var page struct {
				Members []struct {
					ID       string `json:"Id"`
					Created  string `json:"Created"`
					Severity string `json:"Severity"`
					Message  string `json:"Message"`
				} `json:"Members"`
				NextLink string `json:"Members@odata.nextLink"`
			}
			if err := c.getJSON(next, &page); err != nil {
				return Result{}, err
			}
			for _, m := range page.Members {
				e := LogEntry{Log: svc[strings.LastIndex(svc, "/")+1:], ID: m.ID, Created: m.Created, Severity: m.Severity, Message: m.Message}
				if t, err := time.Parse(time.RFC3339, m.Created); err == nil {
					e.created = t
					if !cutoff.IsZero() && t.Before(cutoff) {
						continue
					}
				}
				entries = append(entries, e)
			}
			next = page.NextLink
		}
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].created.After(entries[j].created) })
	total := len(entries)
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	var b strings.Builder
	window := "all time"
	if since > 0 {
		window = "last " + since.String()
	}
	fmt.Fprintf(&b, "%d entries (%s), showing %d", total, window, len(entries))
	for _, e := range entries {
		fmt.Fprintf(&b, "\n%s  %-8s %s", e.Created, orDash(e.Severity), strings.TrimSpace(e.Message))
	}
	return Result{Text: b.String(), Data: map[string]interface{}{"total": total, "entries": entries}}, nil
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
