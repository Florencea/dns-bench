package main

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"

	"github.com/Florencea/dns-bench/internal/config"
)

func openBrowser(targetURL string) error {
	u, err := url.Parse(targetURL)
	if err != nil || u.Scheme != "http" || !config.IsLoopbackHost(u.Hostname()) {
		return fmt.Errorf("refusing to open non-loopback URL %q", targetURL)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u.String())
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u.String())
	case "linux":
		cmd = exec.Command("xdg-open", u.String())
	default:
		return errors.New("unsupported platform for automatic browser launch")
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}
