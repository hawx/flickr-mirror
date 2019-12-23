package main

import (
	_ "github.com/mxk/go-sqlite/sqlite3"
	"hawx.me/code/hadfield"
)

var templates = hadfield.Templates{
	Help: `Usage: flickr-mirror [command] [arguments]

  Something to do with mirroring flickr or something.

  Commands: {{range .}}
    {{.Name | printf "%-15s"}} # {{.Short}}{{end}}
`,
	Command: `Usage: flickr-mirror {{.Usage}}
{{.Long}}
`,
}

var commands = hadfield.Commands{
	cmdGrab,
	cmdIndex,
	cmdServe,
}

func main() {
	hadfield.Run(commands, templates)
}
