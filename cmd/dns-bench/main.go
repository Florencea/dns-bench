package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/Florencea/dns-bench/internal/config"
	"github.com/Florencea/dns-bench/internal/dns"
	"github.com/Florencea/dns-bench/internal/web"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("configuration error", "error", err)
		return 1
	}

	var showVersion bool
	fs := flag.NewFlagSet("dns-bench", flag.ContinueOnError)
	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "Loopback HTTP listen address (default 127.0.0.1:8080, falls back to an ephemeral port if occupied)")
	fs.StringVar(&cfg.DefaultLocale, "locale", cfg.DefaultLocale, "Default UI locale (zh-TW or en-US)")
	fs.BoolVar(&cfg.NoOpen, "no-open", cfg.NoOpen, "Do not open the system web browser automatically on startup")
	fs.BoolVar(&showVersion, "version", false, "Print version information and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	fs.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			cfg.ExplicitAddr = true
		}
	})

	if showVersion {
		fmt.Printf("dns-bench %s (%s %s/%s)\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return 0
	}

	if err := cfg.Validate(); err != nil {
		slog.Error("configuration validation failed", "error", err)
		return 1
	}

	ln, err := listenLoopback(cfg.Addr, cfg.ExplicitAddr)
	if err != nil {
		slog.Error("failed to bind loopback listener", "addr", cfg.Addr, "error", err)
		return 1
	}

	engine := dns.NewEngine(cfg.DNSTimeout, cfg.DoHTimeout)
	srv, err := web.NewServer(cfg, engine)
	if err != nil {
		_ = ln.Close()
		slog.Error("failed to initialize web server", "error", err)
		return 1
	}

	httpSrv := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	url := "http://" + ln.Addr().String() + "/"
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("dns-bench listening", "url", url)
		if serveErr := httpSrv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
		close(errCh)
	}()

	if !cfg.NoOpen {
		if openErr := openBrowser(url); openErr != nil {
			slog.Warn("failed to open default browser automatically", "url", url, "error", openErr)
		}
	}

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutErr := httpSrv.Shutdown(shutdownCtx); shutErr != nil {
			slog.Error("http shutdown error", "error", shutErr)
			return 1
		}
		return 0
	case serveErr := <-errCh:
		if serveErr != nil {
			slog.Error("server error", "error", serveErr)
			return 1
		}
		return 0
	}
}

func listenLoopback(addr string, explicit bool) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		return ln, nil
	}
	if explicit {
		return nil, err
	}
	slog.Warn("default loopback port unavailable, falling back to ephemeral port", "requested", addr, "error", err)
	return net.Listen("tcp", "127.0.0.1:0")
}
