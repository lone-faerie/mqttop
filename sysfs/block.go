package sysfs

import (
	"golang.org/x/sys/unix"
	//	"log"
	"path/filepath"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/file"
	"github.com/lone-faerie/mqttop/procfs"
	"github.com/lone-faerie/mqttop/log"
)

type BlockIO struct {
	old blockIO

	stat string
}

type blockIO struct {
	reads  int64
	writes int64
	ticks  int64
}

func BlockStat(mnt *procfs.Mount) BlockIO {
	var (
		name = filepath.Base(mnt.Dev)
		c    = 0
	)
	for dev := name; len(dev) >= 2; dev = dev[:len(dev)-1] {
		p := Path("block", dev, "stat")
		log.Debug("BlockStat", "Path", p)
		if _, err := file.Stat(p); err == nil && unix.Access(p, unix.R_OK) == nil {
			log.Debug("OK")
			if c == 0 {
				return BlockIO{stat: p}
			}
			pp := Path("block", dev, name, "stat")
			if _, err := file.Stat(pp); err == nil {
				return BlockIO{stat: pp}
			}
			return BlockIO{stat: p}
		} else if mnt.FSType == "zfs" {
			return BlockIO{}
		}
		c++
	}
	return BlockIO{}
}

func (b *BlockIO) IsValid() bool {
	return b.stat != ""
}

func (b *BlockIO) Read() (reads, writes, ticks int64, err error) {
	stat, err := file.Read(b.stat)
	if err != nil {
		return

	}
	var r, w, t []byte
	for i := 0; i < 2; i++ {
		_, stat = byteutil.Column(stat)
	}
	r, stat = byteutil.Column(stat)
	for i := 0; i < 3; i++ {
		_, stat = byteutil.Column(stat)
	}
	w, stat = byteutil.Column(stat)
	for i := 0; i < 2; i++ {
		_, stat = byteutil.Column(stat)
	}
	t, stat = byteutil.Column(stat)

	reads = byteutil.Btoi(r)
	//	log.Println("Reads:", reads)
	oldReads := b.old.reads
	b.old.reads = reads
	reads = (reads - oldReads) * 512
	if reads < 0 {
		reads = 0
	}

	writes = byteutil.Btoi(w)
	//	log.Println("Writes:", writes)
	oldWrites := b.old.writes
	b.old.writes = writes
	writes = (writes - oldWrites) * 512
	if writes < 0 {
		writes = 0
	}

	ticks = byteutil.Btoi(t)
	//	log.Println("Ticks:", ticks)
	oldTicks := b.old.ticks
	b.old.ticks = ticks
	ticks = ticks - oldTicks
	if ticks < 0 {
		ticks = 0
	}

	return
}
