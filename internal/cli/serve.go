package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neolao/batocera-scrap-manager/internal/webui"
)

const serveUsage = `Usage:
  batocera-scrap-manager serve [--addr <host:port>]

Serves the registry's content in a web browser: one page listing every game
grouped by system, and one page per game with its metadata and media. The
served pages can also complete the configured ROMs folders from the registry,
which is what 'scrape' does on the command line.

Options:
  --addr <host:port>   Address to listen on (default 0.0.0.0:8080)

The registry is read once at startup: run 'update' and restart 'serve' to
pick up changes. Press Ctrl+C to stop the server.
`

// defaultServeAddr is the address serve listens on when --addr is not given:
// every interface, so the registry is reachable from another machine on the
// local network.
const defaultServeAddr = "0.0.0.0:8080"

// serveShutdownTimeout bounds how long a graceful shutdown waits for
// in-flight requests (a large media file being downloaded) before giving up.
const serveShutdownTimeout = 5 * time.Second

func runServe(args []string, out io.Writer) int {
	if len(args) > 0 && args[0] == "--help" {
		fmt.Fprint(out, serveUsage)
		return 0
	}

	addr, parsed := parseServeArgs(args, out)
	switch parsed {
	case serveArgsHelpRequested:
		return 0
	case serveArgsInvalid:
		return 1
	}

	cfg, reg, ok := loadConfigAndRegistry(out)
	if !ok {
		return 1
	}

	ctx, stopListeningToSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopListeningToSignals()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}

	fmt.Fprint(out, listeningLines(listener.Addr().String()))

	return serveUntil(ctx, listener, webui.Handler(reg, cfg.RegistryFolder, cfg.RomsFolders), out)
}

// serveArgsOutcome tells runServe what parsing serve's command line
// concluded: start serving, print the usage and stop, or report a mistake.
type serveArgsOutcome int

const (
	serveArgsParsed serveArgsOutcome = iota
	serveArgsHelpRequested
	serveArgsInvalid
)

// parseServeArgs reads serve's options. It writes the usage to out when help
// was asked for (including the flag package's implicit -h), and the problem
// followed by the usage when the command line is malformed.
func parseServeArgs(args []string, out io.Writer) (addr string, outcome serveArgsOutcome) {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() {}
	address := flags.String("addr", defaultServeAddr, "address to listen on")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(out, serveUsage)
			return "", serveArgsHelpRequested
		}
		return failServeArgs(out, "%v", err)
	}
	if flags.NArg() > 0 {
		return failServeArgs(out, "unexpected argument: %s", flags.Arg(0))
	}
	if _, _, err := net.SplitHostPort(*address); err != nil {
		return failServeArgs(out, "invalid address %q: use the host:port form, such as %q",
			*address, ":"+*address)
	}

	return *address, serveArgsParsed
}

// failServeArgs reports a malformed command line: the problem first, then
// the usage, so the user sees what to fix and how.
func failServeArgs(out io.Writer, format string, args ...any) (string, serveArgsOutcome) {
	fmt.Fprintf(out, "error: "+format+"\n", args...)
	fmt.Fprint(out, serveUsage)
	return "", serveArgsInvalid
}

// serveUntil serves handler on listener until ctx is cancelled — which
// happens when the user interrupts the command — then lets in-flight
// requests finish. It returns the exit code: stopping on request is a
// success, so a service manager stopping the process does not read it as a
// crash.
func serveUntil(ctx context.Context, listener net.Listener, handler http.Handler, out io.Writer) int {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		// Per-connection noise (a browser dropping a media download,
		// a port scanner) is not the user's business: it would be
		// interleaved with the command's own output.
		ErrorLog: log.New(io.Discard, "", 0),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), serveShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(out, "error: %v\n", err)
		return 1
	}
	return 0
}

// listeningLines renders what serve prints once it is up: the address it
// actually listens on, and a URL that can be pasted into a browser. A
// wildcard host is not usable as a URL on every platform, so localhost is
// suggested instead.
func listeningLines(address string) string {
	lines := fmt.Sprintf("listening on %s\n", address)

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return lines
	}
	if isWildcardHost(host) {
		host = "localhost"
	}
	return lines + fmt.Sprintf("open http://%s in your browser\n", net.JoinHostPort(host, port))
}

// isWildcardHost reports whether host designates every interface rather than
// one reachable address.
func isWildcardHost(host string) bool {
	return host == "" || host == "0.0.0.0" || host == "::" || host == "[::]"
}
