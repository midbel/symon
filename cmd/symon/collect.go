package main

import (
	"sync"
	"time"

	"github.com/midbel/symon/proc"
)

type Collector struct {
	lastmod time.Time

	mu       sync.RWMutex
	boottime time.Time
	uptime   time.Duration
	process  []proc.ProcInfo
	loadavg  []float64
	swap     proc.MemInfo
	syst     proc.MemInfo
	users    []proc.Who
	conns    []proc.ConnInfo
}

func Monitor() *Collector {
	return &Collector{
		lastmod: time.Now(),
	}
}

func (c *Collector) Run(every time.Duration) {
	c.collect()

	tick := time.NewTicker(every)
	defer tick.Stop()

	for range tick.C {
		c.collect()
	}
}

func (c *Collector) About() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]interface{}{
		"lastmod":     c.lastmod,
		"users":       len(c.users),
		"process":     len(c.process),
		"connections": len(c.conns),
	}
}

func (c *Collector) Conns() []proc.ConnInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conns
}

func (c *Collector) Users() []proc.Who {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.users
}

func (c *Collector) Process() []proc.ProcInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.process
}

func (c *Collector) Free() (proc.MemInfo, proc.MemInfo) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.syst, c.swap
}

func (c *Collector) LoadAvg() []float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.loadavg
}

func (c *Collector) collect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	var wg sync.WaitGroup
	go collect(&wg, func() {
		c.process, _ = proc.Process()
	})
	go collect(&wg, func() {
		c.syst, c.swap, _ = proc.Free()
	})
	go collect(&wg, func() {
		c.users, _ = proc.Current()
	})
	go collect(&wg, func() {
		c.loadavg, _ = proc.LoadAvg()
	})
	go collect(&wg, func() {
		c.boottime, _ = proc.BootTime()
		c.uptime, _ = proc.Uptime()
	})
	go collect(&wg, func() {
		c.conns, _ = proc.Netstat()
	})
	c.lastmod = time.Now()
	wg.Wait()
}

func collect(wg *sync.WaitGroup, do func()) {
	wg.Add(1)
	defer wg.Done()
	do()
}
