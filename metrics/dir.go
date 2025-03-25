package metrics

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/file"
)

type dirEntry struct {
	size   uint64
	parent *dirEntry
	childs []dirEntry
}

type Dir struct {
	Name string
	path string

	dirEntry
	depth    int
	byteSize byteutil.ByteSize

	watched map[string]*dirEntry
	watcher *fsnotify.Watcher

	interval time.Duration
	tick     *time.Ticker
	topic    string

	mu   sync.RWMutex
	once sync.Once
	stop context.CancelFunc
	ch   chan error
}

func NewDir(path string, cfg *config.Config) (*Dir, error) {
	var dcfg *config.DirConfig
	for i := range cfg.Dirs {
		if cfg.Dirs[i].Path == path {
			dcfg = &cfg.Dirs[i]
			break
		}
	}
	if dcfg == nil {
		return nil, errNotSupported(path, ErrDisabled)
	}
	return newDir(dcfg, cfg)
}

func newDir(dcfg *config.DirConfig, cfg *config.Config) (*Dir, error) {
	path := filepath.Clean(dcfg.Path)
	info, err := file.Stat(path)
	if err != nil {
		return nil, errNotSupported(path, err)
	}
	d := &Dir{
		Name: dcfg.FormatName(path),
		path: path,
		dirEntry: dirEntry{
			size: uint64(info.Size()),
		},
		depth: -1,
	}
	if dcfg.Interval > 0 {
		d.interval = dcfg.Interval
	} else {
		d.interval = cfg.Interval
	}
	if dcfg.Topic != "" {
		d.topic = dcfg.Topic
	} else {
		d.topic = "mqttop/metric/dir/" + d.Slug()
	}
	if dcfg.Depth > 0 {
		d.depth = dcfg.Depth
	}
	if !dcfg.Watch {
		d.size = dirSize(path, 0, d.depth)
		d.byteSize = byteSize(dcfg.SizeUnit, d.size)
		return d, nil
	}
	d.watched = map[string]*dirEntry{
		path: &d.dirEntry,
	}
	files, err := file.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() {
			d.init(path+file.Separator+f.Name(), &d.dirEntry, 1)
			continue
		}
		if info, err := f.Info(); err == nil {
			d.size += uint64(info.Size())
		}
	}
	d.byteSize = byteSize(dcfg.SizeUnit, d.size)

	return d, nil
}

func byteSize(s string, b uint64) byteutil.ByteSize {
	size, err := byteutil.ParseSize(s)
	if err != nil {
		return byteutil.SizeOf(b)
	}
	return size
}

func (d *Dir) init(path string, parent *dirEntry, depth int) {
	if depth > d.depth && d.depth > 0 {
		return
	}
	info, err := file.Stat(path)
	if err != nil {
		return
	}
	i := len(parent.childs)
	parent.childs = append(parent.childs, dirEntry{
		size:   uint64(info.Size()),
		parent: parent,
	})
	entry := &parent.childs[i]
	d.watched[path] = entry
	files, err := file.ReadDir(path)
	if err != nil {
		return
	}
	for _, f := range files {
		if f.IsDir() {
			d.init(path+file.Separator+f.Name(), entry, depth+1)
			continue
		}
		if info, err := f.Info(); err == nil {
			entry.size += uint64(info.Size())
		}
	}
	parent.size += entry.size
}

func (d *Dir) Type() string {
	return "dir"
}

func (d *Dir) Topic() string {
	return d.topic
}

func (d *Dir) Slug() string {
	return strings.ReplaceAll(
		strings.TrimPrefix(d.Name, file.Separator),
		file.Separator,
		"_",
	)
}

func (dir *Dir) SetInterval(d time.Duration) {
	dir.mu.Lock()
	if dir.tick != nil && d != dir.interval {
		dir.tick.Reset(d)
	}
	dir.interval = d
	dir.mu.Unlock()
}

func (d *Dir) loopWatch(ctx context.Context) {
	type dirUpdate struct {
		path  string
		event fsnotify.Event
	}
	updates := make(map[string]fsnotify.Op)
	defer d.watcher.Close()
	var (
		err error
		ch  chan error
	)
	select {
	case <-ctx.Done():
		d.Stop()
		return
	case <-d.tick.C:
		d.ch <- nil
	}
	for {
		select {
		case <-ctx.Done():
			d.Stop()
			return
		case e, ok := <-d.watcher.Errors:
			if !ok {
				return
			}
			err = e
			ch = d.ch
		case e, ok := <-d.watcher.Events:
			if !ok {
				return
			}
			path := e.Name
			d.mu.Lock()
			_, ok = d.watched[e.Name]
			if !ok && !file.IsDir(e.Name) {
				e.Op = 0
				path = filepath.Dir(e.Name)
				_, ok = d.watched[path]
			}
			d.mu.Unlock()
			if !ok && !e.Has(fsnotify.Remove) {
				if err := d.add(path); err != nil {
					break
				}
			}
			if _, ok = updates[path]; !ok {
				updates[path] = e.Op
			}
			log.Debug("dir updated", "path", path)
		case <-d.tick.C:
			if len(updates) == 0 {
				break
			}
			d.mu.Lock()
			for path, op := range updates {
				d.update(path, op)
			}
			d.mu.Unlock()
			clear(updates)
			err = nil
			ch = d.ch
		case ch <- err:
			ch = nil
		}
	}
}

func (d *Dir) loop(ctx context.Context) {
	d.mu.Lock()
	d.tick = time.NewTicker(d.interval)
	d.mu.Unlock()
	defer d.tick.Stop()
	defer close(d.ch)
	if d.watcher != nil {
		d.loopWatch(ctx)
		return
	}
	var (
		ch chan error
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.tick.C:
			size := dirSize(d.path, 0, d.depth)
			d.mu.Lock()
			if size != d.size {
				ch = d.ch
			}
			d.size = size
			d.mu.Unlock()
		case ch <- nil:
			ch = nil
		}
	}
}

func (d *Dir) startWatch(ctx context.Context) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	for path := range d.watched {
		w.Add(path)
		log.Debug("Watching dir", "path", path)
	}
	d.watcher = w
	return nil
}

func (d *Dir) Start(ctx context.Context) (err error) {
	if d.interval == 0 {
		log.Warn("Dir interval is 0, not starting", "path", d.path)
		return
	}
	if d.watched != nil {
		if err = d.startWatch(ctx); err != nil {
			return
		}
	}
	d.once.Do(func() {
		ctx, d.stop = context.WithCancel(ctx)
		d.ch = make(chan error)
		go d.loop(ctx)
	})
	return
}

func dirSize(path string, depth, maxDepth int) (size uint64) {
	if depth >= maxDepth && maxDepth > 0 {
		return
	}
	files, err := file.ReadDir(path)
	if err != nil {
		return
	}
	for _, f := range files {
		if f.IsDir() {
			size += dirSize(path+file.Separator+f.Name(), depth+1, maxDepth)
			continue
		}
		if info, err := f.Info(); err == nil {
			size += uint64(info.Size())
		}
	}
	return
}

func hasParent(path, parent string) bool {
	if path == parent {
		return true
	}
	for {
		pathParent := filepath.Dir(path)
		if pathParent == parent {
			return true
		}
		if pathParent == path {
			return false
		}
		path = pathParent
	}
}

func (d *Dir) add(path string) error {
	var (
		parentPath = filepath.Dir(path)
		parent     *dirEntry
	)
	d.mu.Lock()
	for path, dir := range d.watched {
		if hasParent(path, parentPath) {
			parent = dir
			break
		}
	}
	if parent == nil || (d.depth > 0 && parent.depth() > d.depth) {
		d.mu.Unlock()
		return ErrMaxDepth
	}
	i := len(parent.childs)
	parent.childs = append(parent.childs, dirEntry{parent: parent})
	d.watched[path] = &parent.childs[i]
	d.mu.Unlock()

	return d.watcher.Add(path)
}

func (d *dirEntry) depth() int {
	parent := d.parent
	n := 1
	for parent != nil {
		n++
		parent = parent.parent
	}
	return n
}

func (d *Dir) update(path string, op fsnotify.Op) error {
	dir, ok := d.watched[path]
	if !ok {
		return errNotSupported(path, nil)
	}
	if op.Has(fsnotify.Remove) {
		log.Debug("Removing watch", "path", path)
		clear(dir.childs)
		parent := dir.parent
		for parent != nil {
			parent.size -= dir.size
			parent = parent.parent
		}
		delete(d.watched, path)
		return nil
	}
	info, err := file.Stat(path)
	if err != nil {
		return nil
	}
	size := uint64(info.Size())
	files, err := file.ReadDir(path)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if info, err := f.Info(); err == nil {
			size += uint64(info.Size())
		}
	}
	for i := range dir.childs {
		size += dir.childs[i].size
	}
	parent := dir.parent
	for parent != nil {
		parent.size += size - dir.size
		parent = parent.parent
	}
	dir.size = size
	return nil
}

func (d *Dir) updateSlow() error {
	if _, err := file.Stat(d.path); err != nil {
		return err
	}
	size := dirSize(d.path, 0, d.depth)
	if size == d.size {
		return ErrNoChange
	}
	d.size = size
	return nil
}

func (d *Dir) Update() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.watched == nil {
		err := d.updateSlow()
		d.ch <- err
		return err
	}
	for path := range d.watched {
		d.update(path, fsnotify.Write)
	}
	return nil
}

func (d *Dir) Updated() <-chan error {
	return d.ch
}

func (d *Dir) Stop() {
	d.mu.Lock()
	if d.stop != nil {
		d.stop()
	}
	d.mu.Unlock()
}

func (d *Dir) String() string {
	return ""
}

func (d *Dir) AppendText(b []byte) ([]byte, error) {
	d.mu.RLock()
	b = append(b, "{\"path\": \""...)
	b = append(b, d.path...)
	b = append(b, "\", \"size\": "...)
	b = byteutil.AppendSize(b, d.size, d.byteSize)
	b = append(b, '}')
	d.mu.RUnlock()
	return b, nil
}

func (d *Dir) MarshalJSON() ([]byte, error) {
	return d.AppendText(nil)
}
