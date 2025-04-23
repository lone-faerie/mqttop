package metrics

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/file"
)

func fillTestDir(t *testing.T, name string) (size uint64, err error) {
	t.Helper()

	var stat syscall.Statfs_t
	err = syscall.Statfs(name, &stat)
	if err != nil {
		return
	}
	size = uint64(stat.Bsize)

	for i, n := range []uint64{
		100, 1000, 10000, 100000,
	} {
		err = os.WriteFile(filepath.Join(name, "file"+strconv.Itoa(i)), make([]byte, n), 0666)
		if err != nil {
			return
		}
		size += n
	}

	return
}

func testDir(t *testing.T) (*Dir, *config.Config) {
	t.Helper()

	file.SetRoot("/")

	tmp := t.TempDir()
	t.Logf("TempDir: %s", tmp)

	cfg := config.Default()
	cfg.Dirs = append(cfg.Dirs, config.DirConfig{
		MetricConfig: config.MetricConfig{
			Enabled: true,
		},
		Path: tmp,
	})

	dir, err := NewDir(tmp, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if dir == nil {
		t.Fatal("dir is nil")
	}

	return dir, cfg
}

func TestDir(t *testing.T) {
	dir, cfg := testDir(t)

	if want, got := "dir", dir.Type(); got != want {
		t.Errorf("Type: want %q, got %q", want, got)
	}
	topic := cfg.BaseTopic + "/metric/dir/"
	if cfg.Dirs[0].Path[0] == '/' {
		topic += strings.ReplaceAll(cfg.Dirs[0].Path[1:], "/", "_")
	} else {
		topic += strings.ReplaceAll(cfg.Dirs[0].Path, "/", "_")
	}
	if want, got := topic, dir.Topic(); got != want {
		t.Errorf("Topic: want %q, got %q", want, got)
	}
	if want, got := cfg.Interval, dir.interval; got != want {
		t.Errorf("Interval: want %v, got %v", want, got)
	}
}

func TestDir_Update(t *testing.T) {
	dir, _ := testDir(t)

	size, err := fillTestDir(t, dir.path)
	if err != nil {
		t.Fatal(err)
	}

	err = dir.Update()
	if err != nil {
		t.Fatal(err)
	}

	if want, got := size, dir.size; got != want {
		t.Errorf("Size: want %v, got %v", want, got)
	}
}
