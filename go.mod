module github.com/ProCode-Software/klar

go 1.25.0

require github.com/sanity-io/litter v1.5.8

require (
	github.com/ergochat/readline v0.1.3 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/term v0.36.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
)

ignore (
    ./klar-vscode
    ./samples
    ./std
    node_modules
    ./docs
)