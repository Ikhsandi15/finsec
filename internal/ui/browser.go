package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// OpenBrowser opens the target URL in the default cross-platform web browser
func OpenBrowser(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("empty url")
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// rundll32 url.dll,FileProtocolHandler is clean and handles URLs without opening extra cmd windows
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}
