package proc

import (
	"path/filepath"
)

const (
	proc        = "/proc"
	procStatus  = "status"
	procCmdline = "cmdline"
)

var (
	uptimeFile  = filepath.Join(proc, "uptime")
	memFile     = filepath.Join(proc, "meminfo")
	loadavgFile = filepath.Join(proc, "loadavg")
	statFile    = filepath.Join(proc, "stat")
	tcpFile     = filepath.Join(proc, "net", "tcp")
	tcp6File    = filepath.Join(proc, "net", "tcp6")
	udpFile     = filepath.Join(proc, "net", "udp")
	udp6File    = filepath.Join(proc, "net", "udp6")
	routeFile   = filepath.Join(proc, "net", "route")
	wtmpFile    = "/var/log/wtmp"
	utmpFile    = "/var/run/utmp"
)
