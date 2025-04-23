package metrics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/file"
)

func testBattery(t *testing.T) (*Battery, *config.Config) {
	t.Helper()

	err := file.SetRoot("testdata/fixtures")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()

	bat, err := NewBattery(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bat == nil {
		t.Fatal("bat is nil")
	}

	return bat, cfg
}

func TestBattery(t *testing.T) {
	bat, cfg := testBattery(t)

	if want, got := "battery", bat.Type(); got != want {
		t.Errorf("Type: want %q, got %q", want, got)
	}
	if want, got := cfg.Battery.Topic, bat.Topic(); got != want {
		t.Errorf("Topic: want %q, got %q", want, got)
	}
	if want, got := cfg.Interval, bat.interval; got != want {
		t.Errorf("Interval: want %v, got %v", want, got)
	}

	flags := batteryCapacity | batteryEnergy | batteryPower | batteryStatus | batteryVoltage
	if want, got := flags, bat.flags; got != want {
		t.Errorf("Flags: want %v, got %v", want, got)
	}
}

func TestBattery_Update(t *testing.T) {
	bat, _ := testBattery(t)

	err := bat.Update()
	if err != nil {
		t.Fatal(err)
	}

	if want, got := "discharging", bat.status; got != want {
		t.Errorf("Status: want %q, got %q", want, got)
	}
	if want, got := 98, bat.capacity; got != want {
		t.Errorf("Capacity: want %v, got %v", want, got)
	}
	if want, got := int64(4830000), bat.power; got != want {
		t.Errorf("Power: want %v, got %v", want, got)
	}
	if want, got := time.Duration(36857112450000), bat.timeRemaining; got != want {
		t.Errorf("Time Remaining: want %v, got %v", want, got)
	}
}

func TestBattery_MarshalJSON(t *testing.T) {
	bat, _ := testBattery(t)

	data, err := json.Marshal(bat)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"kind":"Li-ion","status":"","capacity":0,"power":0.000000}`

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
