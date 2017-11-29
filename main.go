package main

import (
	_ "github.com/mxk/go-sqlite/sqlite3"
	"hawx.me/code/hadfield"
)

var templates = hadfield.Templates{
	Command: `usage: flickr-mirror [command] [arguments]

  Something to do with mirroring flickr or something.

  Commands: {{range .}}
    {{.Name | printf "%-15s"}} # {{.Short}}{{end}}
`,
	Help: `usage: example {{.Usage}}
{{.Long}}
`,
}

var commands = hadfield.Commands{
	cmdIndex,
	cmdServe,
}

func main() {
	hadfield.Run(commands, templates)
}
