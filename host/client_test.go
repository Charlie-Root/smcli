package host

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Charlie-Root/smcli/config"
)

type fakeBMC struct {
	mu       sync.Mutex
	requests []string
	routes   map[string]string
}

func (f *fakeBMC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	f.mu.Unlock()
	if u, p, ok := r.BasicAuth(); !ok || u != "ADMIN" || p != "secret" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if r.Method != "GET" {
		if strings.HasSuffix(r.URL.Path, "/fail") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"code":"Base.1.4.GeneralError","message":"bad request","@Message.ExtendedInfo":[{"Message":"Property missing","Resolution":"Add it"}]}}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	body, ok := f.routes[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Write([]byte(body))
}

func newTestClient(t *testing.T, routes map[string]string) (*Client, *fakeBMC) {
	t.Helper()
	f := &fakeBMC{routes: routes}
	srv := httptest.NewTLSServer(f)
	t.Cleanup(srv.Close)
	h := config.Host{Name: "h1", BMCAddress: strings.TrimPrefix(srv.URL, "https://"), UsernamePassword: "ADMIN:secret"}
	c, err := New(h, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return c, f
}

var systems = map[string]string{
	"/redfish/v1/Systems": `{"Members":[{"@odata.id":"/redfish/v1/Systems/1"}]}`,
	"/redfish/v1/Systems/1": `{"Manufacturer":"Supermicro","Model":"X","SerialNumber":"S1","BiosVersion":"2.0",
		"PowerState":"On","Status":{"Health":"OK","HealthRollup":"OK"},
		"ProcessorSummary":{"Count":1,"Model":"AMD"},"MemorySummary":{"TotalSystemMemoryGiB":64}}`,
	"/redfish/v1/Managers":   `{"Members":[{"@odata.id":"/redfish/v1/Managers/1"}]}`,
	"/redfish/v1/Managers/1": `{"FirmwareVersion":"1.2.3","Model":"BMC"}`,
}

func TestPowerStatusAndInfo(t *testing.T) {
	c, _ := newTestClient(t, systems)
	r, err := c.PowerStatus()
	if err != nil || r.Text != "power On" {
		t.Fatalf("PowerStatus = %q, %v", r.Text, err)
	}
	r, err = c.Info()
	if err != nil || !strings.Contains(r.Text, "BMC firmware 1.2.3") || !strings.Contains(r.Text, "serial S1") {
		t.Fatalf("Info = %q, %v", r.Text, err)
	}
}

func TestDryRunSendsNoWrite(t *testing.T) {
	c, f := newTestClient(t, systems)
	c.DryRun = true
	r, err := c.Reset("restart")
	if err != nil || !strings.HasPrefix(r.Text, "dry run: would POST /redfish/v1/Systems/1/Actions/ComputerSystem.Reset") {
		t.Fatalf("Reset dry run = %q, %v", r.Text, err)
	}
	for _, req := range f.requests {
		if !strings.HasPrefix(req, "GET ") {
			t.Fatalf("dry run sent %s", req)
		}
	}
}

func TestWriteAndError(t *testing.T) {
	c, f := newTestClient(t, systems)
	if _, err := c.Reset("on"); err != nil {
		t.Fatal(err)
	}
	if got := f.requests[len(f.requests)-1]; got != "POST /redfish/v1/Systems/1/Actions/ComputerSystem.Reset" {
		t.Fatalf("last request = %s", got)
	}
	_, err := c.write("POST", "/fail", map[string]string{}, "")
	if err == nil || !strings.Contains(err.Error(), "HTTP 400: bad request (Property missing Resolution: Add it)") {
		t.Fatalf("error = %v", err)
	}
	if _, err := c.Reset("explode"); err == nil {
		t.Fatal("unknown action accepted")
	}
}

func TestMaskSecrets(t *testing.T) {
	got := MaskSecrets([]byte(`{"UserName":"u","Password":"p","NewPassword":"x"}`))
	if strings.Contains(got, `"p"`) || strings.Contains(got, `"x"`) || !strings.Contains(got, `"u"`) {
		t.Fatalf("MaskSecrets = %s", got)
	}
}

func TestSensors(t *testing.T) {
	routes := map[string]string{
		"/redfish/v1/Chassis": `{"Members":[{"@odata.id":"/redfish/v1/Chassis/1"}]}`,
		"/redfish/v1/Chassis/1/Thermal": `{"Temperatures":[{"Name":"CPU","ReadingCelsius":90,"UpperThresholdCritical":95,"Status":{"Health":"Warning","State":"Enabled"}},
			{"Name":"Gone","Status":{"State":"Absent"}}],"Fans":[{"FanName":"FAN1","Reading":3000,"ReadingUnits":"RPM","Status":{"Health":"OK","State":"Enabled"}}]}`,
		"/redfish/v1/Chassis/1/Power": `{"PowerSupplies":[{"Name":"PSU1","PowerInputWatts":120,"Status":{"Health":"OK","State":"Enabled"}}]}`,
	}
	c, _ := newTestClient(t, routes)
	r, err := c.Sensors(true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.Text, "4 sensors, 1 not OK") || !strings.Contains(r.Text, "! temperature") || strings.Contains(r.Text, "FAN1") {
		t.Fatalf("Sensors = %q", r.Text)
	}
}

func TestSELPagingAndSince(t *testing.T) {
	recent := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	routes := map[string]string{
		"/redfish/v1/Systems":               systems["/redfish/v1/Systems"],
		"/redfish/v1/Systems/1/LogServices": `{"Members":[{"@odata.id":"/redfish/v1/Systems/1/LogServices/Log1"}]}`,
		"/redfish/v1/Systems/1/LogServices/Log1/Entries": `{"Members":[{"Id":"1","Created":"2020-01-01T00:00:00Z","Severity":"OK","Message":"old"}],
			"Members@odata.nextLink":"/redfish/v1/Systems/1/LogServices/Log1/Entries2"}`,
		"/redfish/v1/Systems/1/LogServices/Log1/Entries2": `{"Members":[{"Id":"2","Created":"` + recent + `","Severity":"Critical","Message":"ECC error"}]}`,
	}
	c, _ := newTestClient(t, routes)
	r, err := c.SEL(24*time.Hour, 10, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.Text, "1 entries") || !strings.Contains(r.Text, "ECC error") || strings.Contains(r.Text, "old") {
		t.Fatalf("SEL = %q", r.Text)
	}
	r, _ = c.SEL(0, 0, false)
	if !strings.HasPrefix(r.Text, "2 entries") || strings.Index(r.Text, "ECC error") > strings.Index(r.Text, "old") {
		t.Fatalf("SEL all = %q", r.Text)
	}
}
