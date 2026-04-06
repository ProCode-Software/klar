module github.com/ProCode-Software/klar

go 1.26.0

require (
	github.com/ergochat/readline v0.1.3
	github.com/sanity-io/litter v1.5.8
	golang.org/x/term v0.41.0
	golang.org/x/tools v0.43.0
)

require (
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)

ignore (
	./docs
	./klar-vscode
	./samples
	./std
	node_modules
)
