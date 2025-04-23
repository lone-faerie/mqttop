package metrics

import (
	"encoding/json"
	"math/rand/v2"
	"testing"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/file"
)

func testCPU(t *testing.T) (*CPU, *config.Config) {
	t.Helper()

	err := file.SetRoot("testdata/fixtures")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()

	cpu, err := NewCPU(cfg)

	if err != nil {
		t.Fatal(err)
	}
	if cpu == nil {
		t.Fatal("cpu is nil")
	}

	return cpu, cfg
}

func TestCPU(t *testing.T) {
	cpu, cfg := testCPU(t)

	if want, got := "cpu", cpu.Type(); got != want {
		t.Errorf("Type: want %q, got %q", want, got)
	}
	if want, got := cfg.CPU.Topic, cpu.Topic(); got != want {
		t.Errorf("Topic: want %q, got %q", want, got)
	}
	if want, got := "auto", cpu.selectMode; got != want {
		t.Errorf("Select mode: want %q, got %q", want, got)
	}
	if want, got := cfg.Interval, cpu.interval; got != want {
		t.Errorf("Interval: want %v, got %v", want, got)
	}

	if want, got := cpuTemperature|cpuFrequency|cpuUsage, cpu.flags; got != want {
		t.Errorf("Flags: want %v, got %v", want, got)
	}

	if want, got := "Intel(R) Core(TM) i7-8650U CPU @ 1.90GHz", cpu.Name; got != want {
		t.Errorf("Name: want %q, got %q", want, got)
	}
	if want, got := 8, len(cpu.cores); got != want {
		t.Errorf("Cores: want %v, got %v", want, got)
	}
	if want, got := 7, cpu.cores[7].logical; got != want {
		t.Errorf("Logical: want %v, got %v", want, got)
	}
	if want, got := 4, len(cpu.temps); got != want {
		t.Errorf("Temps: want %v, got %v", want, got)
	}
	if got := cpu.temp; got == nil {
		t.Error("Temp: got nil")
	}
	for i, core := range cpu.cores {
		if want, got := &cpu.temps[core.physical], core.temp; got != want {
			t.Errorf("Core temp %d: want %v, got %v", i, want, got)
		}
		if want, got := int64(2800000), core.freq.Base; got != want {
			t.Errorf("Core base freq %d: want %v, got %v", i, want, got)
		}
		if want, got := int64(3800000), core.freq.Max; got != want {
			t.Errorf("Core max freq %d: want %v, got %v", i, want, got)
		}
		if want, got := int64(800000), core.freq.Min; got != want {
			t.Errorf("Core min freq %d: want %v, got %v", i, want, got)
		}
	}
}

func TestCPU_Update(t *testing.T) {
	cpu, _ := testCPU(t)

	err := cpu.Update()
	if err != nil {
		t.Fatal(err)
	}

	var cores = []struct {
		temp, freq int64
	}{
		{68000, 3124402},
		{71000, 3090417},
		{81000, 800000},
		{67000, 3044179},
		{68000, 2963133},
		{71000, 2882001},
		{81000, 2882023},
		{67000, 2879295},
	}

	temp, freq := cpu.selectFn()
	if want, got := int64(81000), temp; got != want {
		t.Errorf("Temperature: want %v, got %v", want, got)
	}
	if want, got := int64(3124402), freq; got != want {
		t.Errorf("Frequency: want %v, got %v", want, got)
	}
	for i := range cpu.cores {
		if want, got := cores[i].temp, cpu.cores[i].temp.Value(); got != want {
			t.Errorf("Core %d Temperature: want %v, got %v", i, want, got)
		}
		if want, got := cores[i].freq, cpu.cores[i].freq.Curr(); got != want {
			t.Errorf("Core %d Frequency: want %v, got %v", i, want, got)
		}
	}

	if testing.Short() {
		return
	}

	var selects = []struct {
		name string
		fn   func() (temp, freq int64)
		temp int64
		freq int64
	}{
		{"SelectAuto", cpu.SelectAuto, 81000, 3124402},
		{"SelectFirst", cpu.SelectFirst, 68000, 3124402},
		{"SelectAvg", cpu.SelectAvg, 71750, 2708181},
		{"SelectMax", cpu.SelectMax, 81000, 3124402},
		{"SelectMin", cpu.SelectMin, 67000, 800000},
		{"SelectRand", cpu.SelectRand, 81000, 800000}, // cpu.rand.IntN(8) should return 2
	}
	cpu.rand = rand.New(&rand.PCG{})

	for _, s := range selects {
		t.Run(s.name, func(t *testing.T) {
			temp, freq := s.fn()
			if want, got := s.temp, temp; got != want {
				t.Errorf("Temperature: want %v, got %v", want, got)
			}
			if want, got := s.freq, freq; got != want {
				t.Errorf("Frequency: want %v, got %v", want, got)
			}
		})
	}
}

func TestCPU_MarshalJSON(t *testing.T) {
	cpu, _ := testCPU(t)

	data, err := json.Marshal(cpu)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"name":"Intel(R) Core(TM) i7-8650U CPU @ 1.90GHz","temperature":0.000,"frequency":0.000000,"selection_mode":"auto","usage":0,"cores":[{"id":0,"temperature":0.000,"frequency":0.000000,"usage":0},{"id":1,"temperature":0.000,"frequency":0.000000,"usage":0},{"id":2,"temperature":0.000,"frequency":0.000000,"usage":0},{"id":3,"temperature":0.000,"frequency":0.000000,"usage":0},{"id":4,"temperature":0.000,"frequency":0.000000,"usage":0},{"id":5,"temperature":0.000,"frequency":0.000000,"usage":0},{"id":6,"temperature":0.000,"frequency":0.000000,"usage":0},{"id":7,"temperature":0.000,"frequency":0.000000,"usage":0}]}`

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
