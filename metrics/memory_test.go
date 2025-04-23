package metrics

import (
	"encoding/json"
	"testing"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/file"
)

func testMemory(t *testing.T) (*Memory, *config.Config) {
	t.Helper()

	err := file.SetRoot("testdata/fixtures")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()

	mem, err := NewMemory(cfg)

	if err != nil {
		t.Fatal(err)
	}
	if mem == nil {
		t.Fatal("mem is nil")
	}

	return mem, cfg
}

func TestMemory(t *testing.T) {
	mem, cfg := testMemory(t)

	if want, got := "memory", mem.Type(); got != want {
		t.Errorf("Type: want %q, got %q", want, got)
	}
	if want, got := cfg.Memory.Topic, mem.Topic(); got != want {
		t.Errorf("Topic: want %q, got %q", want, got)
	}
	if want, got := cfg.Interval, mem.interval; got != want {
		t.Errorf("Interval: want %v, got %v", want, got)
	}

	if want, got := uint64(16042172416), mem.total; got != want {
		t.Errorf("Total: want %v, got %v", want, got)
	}
	if want, got := uint64(1023406080), mem.swapTotal; got != want {
		t.Errorf("Swap Total: want %v, got %v", want, got)
	}
	if want, got := byteutil.GiB, mem.size; got != want {
		t.Errorf("Size: want %v, got %v", want, got)
	}
	if want, got := byteutil.MiB, mem.swapSize; got != want {
		t.Errorf("Swap Size: want %v, got %v", want, got)
	}
}

func TestMemory_Update(t *testing.T) {
	mem, _ := testMemory(t)

	err := mem.Update()
	if err != nil {
		t.Fatal(err)
	}

	if want, got := uint64(450891776), mem.free; got != want {
		t.Errorf("Free: want %v, got %v", want, got)
	}
	if want, got := uint64(12295823360), mem.cached; got != want {
		t.Errorf("Cached: want %v, got %v", want, got)
	}
	avail := uint64(12746715136) // free + cached
	if want, got := avail, mem.avail; got != want {
		t.Errorf("Available: want %v, got %v", want, got)
	}
	used := uint64(3295457280) // total - avail
	if want, got := used, mem.used; got != want {
		t.Errorf("Used: want %v, got %v", want, got)
	}

	if want, got := uint64(1023406080), mem.swapTotal; got != want {
		t.Errorf("Swap Total: want %v, got %v", want, got)
	}
	if want, got := uint64(315383808), mem.swapFree; got != want {
		t.Errorf("Swap Free: want %v, got %v", want, got)
	}
	used = uint64(708022272) // swapTotal - swapFree
	if want, got := used, mem.swapUsed; got != want {
		t.Errorf("Swap Used: want %v, got %v", want, got)
	}
}

func TestMemory_MarshalJSON(t *testing.T) {
	mem, _ := testMemory(t)

	data, err := json.Marshal(mem)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"total":14.940,"used":0,"available":0,"cached":0,"free":0,"swapTotal":975.996,"swapUsed":0,"swapFree":0}`

	if got := string(data); got != want {
		var i int
		for i := range got {
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
