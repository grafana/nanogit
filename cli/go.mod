module github.com/grafana/nanogit/cli

go 1.24.2

require (
	github.com/fatih/color v1.18.0
	github.com/grafana/nanogit v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.8.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/grafana/nanogit => ..
