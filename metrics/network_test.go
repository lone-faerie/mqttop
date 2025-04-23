package metrics

import (
	"encoding/json"
	stdnet "net"
	"net/netip"
	"testing"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/file"
)

func testNet(t *testing.T) (*Net, *config.Config) {
	t.Helper()

	err := file.SetRoot("testdata/fixtures")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()

	net, err := NewNet(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if net == nil {
		t.Fatal("net is nil")
	}

	return net, cfg
}

func maybeGetAddr(t *testing.T, name string) netip.Addr {
	t.Helper()

	iface, err := stdnet.InterfaceByName(name)
	if err != nil {
		return netip.Addr{}
	}
	addrs, err := iface.Addrs()
	if err != nil || len(addrs) == 0 {
		return netip.Addr{}
	}
	s := addrs[0].String()
	if a, err := netip.ParseAddr(s); err == nil {
		return a
	}
	if ap, err := netip.ParseAddrPort(s); err == nil {
		return ap.Addr()
	}
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Addr()
	}
	return netip.Addr{}
}

func TestNet(t *testing.T) {
	net, cfg := testNet(t)

	if want, got := "net", net.Type(); got != want {
		t.Errorf("Type: want %q, got %q", want, got)
	}
	if want, got := cfg.Net.Topic, net.Topic(); got != want {
		t.Errorf("Topic: want %q, got %q", want, got)
	}
	if want, got := cfg.Interval, net.interval; got != want {
		t.Errorf("Interval: want %v, got %v", want, got)
	}

	if want, got := 1, len(net.interfaces); got != want {
		t.Fatalf("Interfaces: want %v, got %v", want, got)
	}
	if want, got := "eth0", net.interfaces["eth0"].name; got != want {
		t.Errorf("Name: want %q, got %q", want, got)
	}
	if want, got := maybeGetAddr(t, "eth0"), net.interfaces["eth0"].ip; got != want {
		t.Errorf("Address: want %v, got %v", want, got)
	}
	if want, got := byteutil.MiBps, net.interfaces["eth0"].rate; got != want {
		t.Errorf("Rate: want %v, got %v", want, got)
	}
}

func TestNet_Update(t *testing.T) {
	net, _ := testNet(t)

	err := net.Update()
	if err != nil {
		t.Fatal(err)
	}

	if want, got := uint64(116706680863), net.interfaces["eth0"].rx; got != want {
		t.Errorf("Rx: want %v, got %v", want, got)
	}
	if want, got := uint64(145311386254), net.interfaces["eth0"].tx; got != want {
		t.Errorf("Tx: want %v, got %v", want, got)
	}
}

func TestNet_MarshalJSON(t *testing.T) {
	net, _ := testNet(t)

	net.interfaces["eth0"].ip = netip.Addr{}

	data, err := json.Marshal(net)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"eth0":{"running":false}}`

	if got := string(data); got != want {
		var i int
		for i = range got {
			if i >= len(want) {
				i = len(want) - 1
				break
			}
			if got[i] != want[i] {
				break
			}
		}
		t.Errorf("result differs at char %d\nwant %q\ngot  %q", i, want[:i+1], got[:i+1])
	}
}
