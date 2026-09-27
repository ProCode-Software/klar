package docs

import (
	"net"
	"net/http"

	"github.com/ProCode-Software/klar/internal/cli"
	"github.com/ProCode-Software/klar/internal/cli/ansi"
)

func startServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Klar documentation"))
	})

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		cli.Failure("Failed to start HTTP server:", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	ansi.TagPrintfln(
		"<g!>📖 You can now view the documentation in your browser at <b!>http://localhost:%[1]d</b!>!</> "+
			"<d>Press <w!>Ctrl+C</w!> to stop</d>", port,
	)

	http.Serve(listener, nil)
}
