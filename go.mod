module github.com/ProCode-Software/klar

go 1.26.0

require (
	github.com/ergochat/readline v0.1.3
	github.com/sanity-io/litter v1.5.8
	golang.org/x/sync v0.21.0
	golang.org/x/term v0.44.0
	golang.org/x/tools v0.46.0
)

require (
	github.com/aclements/go-moremath v0.0.0-20241023150245-c8bbc672ef66 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/perf v0.0.0-20260610192853-712aea8b4705 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
)

ignore (
	./docs
	./klar-vscode
	./samples
	./std
	node_modules
)

tool (
	golang.org/x/perf/cmd/benchstat
	golang.org/x/tools/cmd/stringer
)
