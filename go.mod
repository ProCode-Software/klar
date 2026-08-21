module github.com/ProCode-Software/klar

go 1.27.0

require (
	github.com/ergochat/readline v0.1.3
	github.com/sanity-io/litter v1.5.8
	golang.org/x/sync v0.22.0
	golang.org/x/term v0.45.0
	golang.org/x/tools v0.49.0
)

require (
	github.com/aclements/go-moremath v0.0.0-20241023150245-c8bbc672ef66 // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/perf v0.0.0-20260819171926-ebcb4798430d // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

ignore (
	./docs
	./klar-vscode
	./packages
	./samples
	./std
	node_modules
)

tool (
	golang.org/x/perf/cmd/benchstat
	golang.org/x/tools/cmd/stringer
)
