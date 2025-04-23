//go:build !nogpu

package metrics

import (
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/file"
)

func testNvidiaGPU(t *testing.T) (*NvidiaGPU, *config.Config) {
	t.Helper()

	err := file.SetRoot("/")
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()

	gpu, err := NewNvidiaGPU(cfg)
	if err == nvml.ERROR_LIBRARY_NOT_FOUND {
		t.Skip("nvml library not found")
	} else if err != nil {
		t.Fatal(err)
	}
	if gpu == nil {
		t.Fatal("bat is nil")
	}

	return gpu, cfg
}

func TestNvidiaGPU(t *testing.T) {
	gpu, cfg := testNvidiaGPU(t)
	t.Cleanup(func() { gpu.Stop() })

	if want, got := "gpu", gpu.Type(); got != want {
		t.Errorf("Type: want %q, got %q", want, got)
	}
	if want, got := cfg.GPU.Topic, gpu.Topic(); got != want {
		t.Errorf("Topic: want %q, got %q", want, got)
	}
	if want, got := cfg.Interval, gpu.interval; got != want {
		t.Errorf("Interval: want %v, got %v", want, got)
	}
}
