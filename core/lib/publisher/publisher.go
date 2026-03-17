package publisher

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Config holds publisher configuration
type Config struct {
	SiteDir string
}

// Result holds publisher output
type Result struct {
	Success bool
}

// Publish deploys the site using lightspeed deploy
func Publish(cfg Config) (*Result, error) {
	// Check that lightspeed is on PATH
	lightspeedPath, err := exec.LookPath("lightspeed")
	if err != nil {
		return nil, fmt.Errorf("lightspeed CLI not found on PATH — install it first")
	}
	_ = lightspeedPath

	// Check that site directory exists
	if _, err := os.Stat(cfg.SiteDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("site directory does not exist: %s", cfg.SiteDir)
	}

	// Check for site.properties
	propsPath := cfg.SiteDir + "/site.properties"
	if _, err := os.Stat(propsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("site.properties not found in %s", cfg.SiteDir)
	}

	// Run lightspeed deploy
	cmd := exec.Command("lightspeed", "deploy")
	cmd.Dir = cfg.SiteDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			return nil, fmt.Errorf("lightspeed CLI not found — install it first")
		}
		return nil, fmt.Errorf("lightspeed deploy failed: %w", err)
	}

	return &Result{Success: true}, nil
}
