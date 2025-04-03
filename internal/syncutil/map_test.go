package syncutil_test

import (
	"sync"
	"testing"

	"github.com/lone-faerie/mqttop/internal/syncutil"
)

var tests = []struct {
	Key   string
	Val   bool
	Store bool
	Load  bool
}{
	{"key1", true, true, true},
	{"key2", true, false, true},
	{"key3", false, true, true},
	{"key4", true, false, true},
	{"key5", false, true, true},
	{"key6", true, true, true},
	{"key7", true, false, true},
	{"key8", false, true, true},
	{"key9", true, false, true},
	{"key10", false, true, true},
	{"key11", true, true, true},
	{"key12", true, false, true},
	{"key13", false, true, true},
	{"key14", true, false, true},
	{"key15", false, true, true},
}

func BenchmarkSyncMap(b *testing.B) {
	for b.Loop() {
		var (
			m  sync.Map
			wg sync.WaitGroup
		)
		for _, tt := range tests {
			if !tt.Store {
				continue
			}
			wg.Add(1)
			go func(key string, val bool) {
				m.Store(key, val)
				wg.Done()
			}(tt.Key, tt.Val)
		}
		wg.Wait()
		var v, ok bool
		var i interface{}
		for _, tt := range tests {
			if !tt.Load {
				continue
			}
			i, ok = m.Load(tt.Key)
			if !ok {
				continue
			}
			v = i.(bool)
			_ = v
		}
	}
}

func BenchmarkMap(b *testing.B) {
	for b.Loop() {
		var (
			m  syncutil.Map[string, bool]
			wg sync.WaitGroup
		)
		m.Make()
		for _, tt := range tests {
			if !tt.Store {
				continue
			}
			wg.Add(1)
			go func(key string, val bool) {
				m.Store(key, val)
				wg.Done()
			}(tt.Key, tt.Val)
		}
		wg.Wait()
		var v, ok bool
		for _, tt := range tests {
			v, ok = m.Load(tt.Key)
			if !ok {
				continue
			}
			_ = v
		}
	}
}
