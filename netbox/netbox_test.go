package netbox

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func fakeNetBox(t *testing.T) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/dcim/devices/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token tok" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		q := r.URL.Query()
		switch {
		case q.Get("name") == "srv1":
			w.Write([]byte(`{"next":null,"results":[{"id":1,"name":"srv1","oob_ip":{"id":10}}]}`))
		case q.Get("name") == "nooob":
			w.Write([]byte(`{"next":null,"results":[{"id":2,"name":"nooob","oob_ip":null}]}`))
		case q.Get("rack_id") == "5" && q.Get("offset") == "":
			w.Write([]byte(`{"next":"http://` + r.Host + `/api/dcim/devices/?rack_id=5&offset=1","results":[{"id":1,"name":"srv1","oob_ip":{"id":10}}]}`))
		case q.Get("rack_id") == "5":
			w.Write([]byte(`{"next":null,"results":[{"id":2,"name":"nooob","oob_ip":null}]}`))
		default:
			w.Write([]byte(`{"next":null,"results":[]}`))
		}
	})
	mux.HandleFunc("/api/dcim/racks/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") == "A.R10" {
			w.Write([]byte(`{"next":null,"results":[{"id":5}]}`))
			return
		}
		w.Write([]byte(`{"next":null,"results":[]}`))
	})
	mux.HandleFunc("/api/ipam/ip-addresses/10/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"address":"192.0.2.10/24","assigned_object_type":"dcim.interface","assigned_object_id":100}`))
	})
	mux.HandleFunc("/api/dcim/interfaces/100/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"IPMI","custom_fields":{"ipmi_username":"ADMIN","ipmi_password":"secret"}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return New(srv.URL, "tok", 5*time.Second)
}

func TestResolveByName(t *testing.T) {
	c := fakeNetBox(t)
	got, err := c.Resolve(Filter{Names: []string{"srv1", "nooob", "missing"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d targets", len(got))
	}
	if got[0].Err != nil || got[0].Host.BMCAddress != "192.0.2.10" || got[0].Host.UsernamePassword != "ADMIN:secret" {
		t.Fatalf("srv1 = %+v", got[0])
	}
	if got[1].Err == nil || got[1].Err.Error() != "no OOB IP set in NetBox" {
		t.Fatalf("nooob err = %v", got[1].Err)
	}
	if got[2].Err == nil || got[2].Host.Name != "missing" {
		t.Fatalf("missing = %+v", got[2])
	}
}

func TestResolveByRackPaginates(t *testing.T) {
	c := fakeNetBox(t)
	got, err := c.Resolve(Filter{Rack: "A.R10"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Host.Name != "srv1" || got[1].Host.Name != "nooob" {
		t.Fatalf("got %+v", got)
	}
	if _, err := c.Resolve(Filter{Rack: "nope"}); err == nil {
		t.Fatal("unknown rack accepted")
	}
}
