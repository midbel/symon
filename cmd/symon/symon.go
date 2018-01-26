package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
		Usage: "meminfo",
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
		Usage: "netstat",
		Short: "print information about active connections on a system",
		Run:   runNetstat,
	},
	{
		Usage: "status",
		Short: "print statistics about system status from boot time",
		Run:   runStat,
	},
	{
		Usage: "load [-e] [-w]",
		Short: "print information about cpu usage from boot time",
		Run:   runPercent,
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

func runPercent(cmd *cli.Command, args []string) error {
	every := cmd.Flag.Duration("e", time.Second, "every")
	watch := cmd.Flag.Bool("w", false, "watch")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	for p := range symon.TotalPercentCPU(*every) {
		fmt.Fprintf(os.Stdout, "CPU usage: %.2f%%", p)
		if !*watch {
			fmt.Fprintln(os.Stdout)
			break
		}
		fmt.Fprint(os.Stdout, "\r")
	}
	return nil
}

func runStat(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	const pattern = "%-5s %6.2f %6.2f %6.2f %6.2f %6.2f"

	s, err := symon.Stat()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 2, 2, '\t', tabwriter.AlignRight)
	log.SetOutput(w)

	cs := make([]*symon.Core, 0, 1+len(s.Cores))
	cs = append(cs, s.Main)
	cs = append(cs, s.Cores...)
	log.Printf("%5s %6s %6s %6s %6s %6s", " ", "user", "syst", "nice", "idle", "wait")
	for _, c := range cs {
		log.Printf(pattern, "%"+c.Label, c.User, c.Syst, c.UserN, c.Idle, c.Wait)
	}
	log.Println()
	log.Printf("boot %s (%s)", s.Boot.Format(time.RFC1123), time.Now().Format(time.RFC1123))
	log.Printf("running  %d", s.Running)
	log.Printf("waiting  %d", s.Waiting)

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
	for _, r := range rs {
		log.Printf("%+v", r)
	}
	return nil
}

func runNetstat(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	cs, err := symon.Netstat()
	if err != nil {
		return err
	}
	for _, c := range cs {
		log.Printf("%+v", c)
	}
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
	const pattern = "%-6s %9.2f %9.2f %9.2f %9.2f %9.2f %9.2f"

	size := symon.Kilo
	cmd.Flag.Var(&size, "s", "unit size")
	watch := cmd.Flag.Bool("w", false, "")
	total := cmd.Flag.Bool("t", false, "")
	every := cmd.Flag.Duration("e", time.Second, "")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 12, 2, 2, '\t', tabwriter.AlignRight)
	log.SetOutput(w)

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
		fmt.Fprintf(os.Stdout, "%6s %9s %9s %9s %9s %9s %9s", " ", "total", "used", "free", "shared", "buf/cache", "available")
		fmt.Fprintln(os.Stdout)

		var n symon.M
		for _, m := range ms {
			z := m.Scale(size)
			fmt.Fprintf(os.Stdout, pattern, z.Device+":", z.Total, z.Used(), z.Free, z.Share, z.Cache+z.Buffers, z.Available)
			fmt.Fprintln(os.Stdout)
			if *total {
				n = n.Cumulate(z)
			}
		}
		if *total {
			n.Device = "total"
			fmt.Fprintf(os.Stdout, pattern, n.Device+":", n.Total, n.Used(), n.Free, n.Share, n.Cache+n.Buffers, n.Available)
			fmt.Fprintln(os.Stdout)
		}
		if !*watch {
			return nil
		}
		<-time.After(*every)
		fmt.Fprint(os.Stdout, "\033[H\033[2J")
	}
	return nil
}

func runWho(cmd *cli.Command, args []string) error {
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
	for _, u := range us {
		log.Printf("%+v", u)
	}
	return nil
}

func runServe(cmd *cli.Command, args []string) error {
	addr := cmd.Flag.String("a", ":9090", "bind to address")
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	http.Handle("/", rest.Version())
	http.Handle("/stats/", rest.Stat())
	http.Handle("/mount/", rest.Mount())
	http.Handle("/routes/", rest.Routes())
	http.Handle("/netstat/", rest.Netstat())
	http.Handle("/version/", rest.Version())
	http.Handle("/meminfo/", rest.Free())
	http.Handle("/users/", rest.Who())
	http.Handle("/process/", rest.Process())

	return http.ListenAndServe(*addr, nil)
}
