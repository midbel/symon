package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/midbel/cli"
	"github.com/midbel/symon"
	"github.com/midbel/symon/rest"
)

const helpText = `{{.Name}} contains various actions to monitor system activities.

Usage:

  {{.Name}} command [arguments]

The commands are:

{{range .Commands}}{{printf "  %-9s %s" .String .Short}}
{{end}}

Use {{.Name}} [command] -h for more information about its usage.
`

var commands = []*cli.Command{
	{
		Usage: "free",
		Short: "display amount of memory used in the system",
		Run:   runMem,
	},
	{
		Usage: "serve",
		Short: "run a webserver",
		Run:   runServe,
	},
	{
		Usage: "who",
		Short: "print information about who are logged in",
		Run:   runWho,
	},
	{
		Usage: "version",
		Short: "print version information",
		Run:   runVersion,
	},
	{
		Usage: "routes",
		Short: "print routes known by a system",
		Run:   runRoutes,
	},
	{
		Usage: "interfaces",
		Short: "print network interfaces known by a system",
		Run:   runInterfaces,
	},
	{
		Usage: "links",
		Short: "print links known by a system",
		Alias: []string{"arp"},
		Run:   runLinks,
	},
	{
		Usage: "netstat",
		Short: "print information about active connections on a system",
		Run:   runNetstat,
	},
	{
		Usage: "process",
		Short: "print process currently running on a system",
		Run:   runProcess,
	},
	{
		Usage: "lastlog",
		Short: "print the more recent login of users known by the system",
		Run:   runLastlog,
	},
	{
		Usage: "load",
		Short: "print cpu usage percentage",
		Run:   runPercents,
	},
}

func main() {
	log.SetFlags(0)
	usage := func() {
		data := struct {
			Name     string
			Commands []*cli.Command
		}{
			Name:     filepath.Base(os.Args[0]),
			Commands: commands,
		}
		t := template.Must(template.New("help").Parse(helpText))
		t.Execute(os.Stderr, data)

		os.Exit(2)
	}
	if err := cli.Run(commands, usage, nil); err != nil {
		log.Fatalln(err)
	}
}

func runPercents(cmd *cli.Command, args []string) error {
	e := cmd.Flag.Duration("e", time.Millisecond*100, "")
	g := cmd.Flag.Bool("g", false, "")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	us, err := symon.Percents(*e)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 2, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "name\tuser\tsyst\tnice\tidle\twait\ttotal")

	const pattern = "%s\t%5.2f\t%5.2f\t%5.2f\t%5.2f\t%5.2f\t%5.2f\n"
	for i, u := range us {
		if !*g && i == 0 {
			continue
		}
		fmt.Fprintf(w, pattern, u.Label, u.User, u.Syst, u.UserN, u.Idle, u.Wait, u.Total)
	}
	return nil
}

func runLastlog(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	as, err := symon.Last()
	if err != nil {
		return err
	}
	sort.Slice(as, func(i, j int) bool {
		return as[i].Uid < as[j].Uid
	})
	w := tabwriter.NewWriter(os.Stdout, 12, 2, 2, ' ', 0)
	defer w.Flush()
	for _, a := range as {
		if !a.Found() {
			fmt.Fprintf(w, "%s\t%s\t%s\n", a.User(), "-", "**never logged in**")
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", a.User(), a.Line, a.When)
	}
	return nil
}

func runProcess(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	const pattern = "%s\t%d\t%d\t%s\t%.2f\t%.2f\t%s\t%s\t%s\n"
	w := tabwriter.NewWriter(os.Stdout, 12, 2, 2, ' ', 0)
	defer w.Flush()

	ps, err := symon.Process()
	if err != nil {
		return err
	}
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].Pid < ps[j].Pid
	})
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "USER", "PID", "PPID", "SINCE", "%CPU", "%MEM", "TTY", "STAT", "CMD")
	for _, p := range ps {
		fmt.Fprintf(w, pattern, p.User(), p.Pid, p.Parent, formatDuration(p.Uptime), p.Core, 0.0, "?", p.State, p.Command())
	}
	return nil
}

func runRoutes(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	rs, err := symon.Routes()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 2, 2, ' ', 0)
	const pattern = "%s\t%s\t%s\t%s\t\n"
	fmt.Fprintf(w, pattern, "destination", "gateway", "mask", "interface")
	for _, r := range rs {
		fmt.Fprintf(w, pattern, r.Address, r.Gateway, r.Mask, r.Interface)
	}
	w.Flush()
	return nil
}

func runInterfaces(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	fs, err := symon.Interfaces()
	if err != nil {
		return err
	}
	for _, i := range fs {
		log.Printf("%+v", i)
	}
	return nil
}

func runLinks(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	ls, err := symon.Links()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 2, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "address", "type", "mac", "mask", "interface")
	for _, l := range ls {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", l.Address, l.Type, l.Hardware, l.Mask, l.Interface)
	}
	return nil
}

func runNetstat(cmd *cli.Command, args []string) error {
	var ls symon.Layers
	cmd.Flag.Var(&ls, "p", "protocol")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	cs, err := symon.Netstat([]string(ls)...)
	if err != nil {
		return err
	}
	const pattern = "%s\t%d\t%d\t%s\t%s\t%s\t%s\t%s\n"

	w := tabwriter.NewWriter(os.Stdout, 12, 2, 2, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", "proto", "recv", "send", "local", "remote", "state", "user", "pid/cmd")
	for _, c := range cs {
		fmt.Fprintf(w, pattern, c.Proto, c.Recv, c.Send, c.Local, c.Remote, c.Status(), c.User(), c.Command)
	}
	w.Flush()
	return nil
}

func runVersion(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	w, d := symon.Uptime()
	i, v, err := symon.Version()
	if err != nil {
		return err
	}
	log.Printf("%s (%s) - %s (%s)", v, i, w.Format(time.RFC1123), d)
	return nil
}

func runMem(cmd *cli.Command, args []string) error {
	const pattern = "%-6s\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t%.2f\t\n"

	size := symon.Kilo
	cmd.Flag.Var(&size, "s", "unit size")
	watch := cmd.Flag.Bool("w", false, "")
	total := cmd.Flag.Bool("t", false, "")
	every := cmd.Flag.Duration("e", time.Second, "")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 9, 2, 4, ' ', 0)

	if *every <= 0 {
		*every = time.Second
	}

	if *watch {
		fmt.Fprint(os.Stdout, "\033[H\033[2J")
	}
	for {
		ms, err := symon.Free()
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "%-6s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", "dev", "total", "used", "free", "shared", "cached", "avail")

		var n symon.M
		for _, m := range ms {
			z := m.Scale(size)
			fmt.Fprintf(w, pattern, z.Device, z.Total, z.Used(), z.Free, z.Share, z.Cache+z.Buffers, z.Available)
			if *total {
				n = n.Cumulate(z)
			}
		}
		if *total {
			fmt.Fprintf(w, pattern, "total", n.Total, n.Used(), n.Free, n.Share, n.Cache+n.Buffers, n.Available)
		}
		w.Flush()
		if !*watch {
			return nil
		}
		fmt.Fprint(os.Stdout, "\033[H\033[2J")
		<-time.After(*every)
	}
	return nil
}

func runWho(cmd *cli.Command, args []string) error {
	const pattern = "%s\t%s\t%s\t%s\t%s\t%s\t\n"
	all := cmd.Flag.Bool("a", false, "all")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	var (
		us  []symon.U
		err error
	)
	if !*all {
		us, err = symon.Utmp()
	} else {
		us, err = symon.Wtmp()
	}
	if err != nil {
		return err
	}
	sort.Slice(us, func(i, j int) bool {
		return us[i].Pid < us[j].Pid && us[i].Seconds < us[j].Seconds
	})

	w := tabwriter.NewWriter(os.Stdout, 9, 2, 4, ' ', 0)
	fmt.Fprintf(w, pattern, "user", "tty", "origin", "at", "idle", "command")
	for _, u := range us {
		t := u.Since()
		s := t.Format("2006-01-02 15:04")
		d := time.Since(t)
		fmt.Fprintf(w, pattern, u.User, u.Id, u.Hostname(), s, formatDuration(d), u.Command())
	}
	w.Flush()
	return nil
}

func runServe(cmd *cli.Command, args []string) error {
	addr := cmd.Flag.String("a", ":9090", "bind to address")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	http.Handle("/", rest.Version())
	http.Handle("/mount/", rest.Mount())
	http.Handle("/interfaces/", rest.Interfaces())
	http.Handle("/version/", rest.Version())
	http.Handle("/meminfo/", rest.Free())
	http.Handle("/users/", rest.Who())
	http.Handle("/process/", rest.Process())
	http.Handle("/stats/", rest.Stats())

	return http.ListenAndServe(*addr, nil)
}

func formatDuration(d time.Duration) string {
	z := d.Minutes()
	h, m := int(z)/60, int(z)%60
	return fmt.Sprintf("%dh%02dm", h, m)
}
