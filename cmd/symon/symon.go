package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

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

func runMem(cmd *cli.Command, args []string) error {
	if err := cmd.Flag.Parse(args); err != nil {
		return err
	}
	ms, err := symon.Free()
	if err != nil {
		return err
	}
	for _, m := range ms {
		log.Printf("%v", m)
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
	http.Handle("/meminfo/", rest.Free())
	http.Handle("/users/", rest.Who())
	http.Handle("/process/", rest.Process())

	return http.ListenAndServe(*addr, nil)
}
