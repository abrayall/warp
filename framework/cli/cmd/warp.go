package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"warp/core/lib/modernizer"
	"warp/core/lib/scraper"
	"warp/core/lib/ui"

	"github.com/spf13/cobra"
)

var (
	warpName         string
	warpDir          string
	warpSkipScrape   bool
	warpSkipDeploy   bool
	warpSkipModern   bool
	warpTimeout      int
	warpURL          string
	warpInstructions string
)

func runWarp(cmd *cobra.Command, args []string) {
	// Determine URL
	targetURL := warpURL
	if len(args) > 0 {
		targetURL = args[0]
	}
	if targetURL == "" {
		cmd.Help()
		return
	}

	// Normalize URL
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	// Derive site name
	name := warpName
	if name == "" {
		name = deriveNameFromURL(targetURL)
	}

	// Print header
	ui.PrintHeader(Version)
	ui.PrintKeyValue("URL", targetURL)
	ui.PrintKeyValue("Name", name)
	fmt.Println()

	// Set up working directories
	baseDir := filepath.Join(warpDir, name)
	scrapedDir := filepath.Join(baseDir, "scraped")
	siteDir := filepath.Join(baseDir, "site")

	// If not skipping anything and the directory already exists, back it up
	if !warpSkipScrape && !warpSkipModern && !warpSkipDeploy {
		if _, err := os.Stat(baseDir); err == nil {
			backupDir := findBackupName(baseDir)
			ui.PrintInfo("Backing up previous run to %s", backupDir)
			if err := os.Rename(baseDir, backupDir); err != nil {
				ui.PrintWarning("Could not backup: %s", err)
			}
			fmt.Println()
		}
	}

	// Phase 1: Scrape
	if !warpSkipScrape {
		fmt.Println(ui.Header("Phase 1: Scraping website"))
		fmt.Println()

		scrapeResult, err := scraper.Scrape(scraper.Config{
			URL:       targetURL,
			OutputDir: scrapedDir,
			Timeout:   time.Duration(warpTimeout) * time.Second,
		})
		if err != nil {
			ui.PrintError("Scraping failed: %s", err)
			os.Exit(1)
		}

		ui.PrintSuccess("Scraped %s", scrapeResult.Title)
		ui.PrintKeyValue("  Output", scrapeResult.OutputDir)
		ui.PrintKeyValue("  Assets", fmt.Sprintf("%d files", len(scrapeResult.Assets)))
		fmt.Println()
	} else {
		ui.PrintWarning("Skipping scrape (reusing existing)")
		fmt.Println()
	}

	// Phase 2: Modernize
	if !warpSkipModern {
		fmt.Println(ui.Header("Phase 2: Modernizing with Claude"))
		fmt.Println()

		modernResult, err := modernizer.Modernize(modernizer.Config{
			Name:         name,
			BaseDir:      baseDir,
			ScrapedDir:   scrapedDir,
			OutputDir:    siteDir,
			Instructions: warpInstructions,
		})
		if err != nil {
			ui.PrintError("Modernization failed: %s", err)
			os.Exit(1)
		}

		ui.PrintSuccess("Modernized site created")
		ui.PrintKeyValue("  Output", modernResult.SiteDir)
		fmt.Println()
	} else {
		ui.PrintWarning("Skipping modernization")
		fmt.Println()
	}

	// Phase 3: Preview with lightspeed start
	if !warpSkipDeploy {
		fmt.Println(ui.Header("Phase 3: Starting local preview"))
		fmt.Println()

		sp := ui.NewSpinner("Starting lightspeed server...")

		// Stop any existing container first
		stopCmd := exec.Command("lightspeed", "stop")
		stopCmd.Dir = siteDir
		stopCmd.CombinedOutput() // ignore errors

		startCmd := exec.Command("lightspeed", "start")
		startCmd.Dir = siteDir

		output, err := startCmd.CombinedOutput()
		sp.Finish()

		outputStr := string(output)

		if err != nil {
			// Check if it's just "already running"
			if strings.Contains(outputStr, "already running") {
				ui.PrintWarning("Server already running")
			} else {
				ui.PrintError("Preview failed: %s", err)
				os.Exit(1)
			}
		}

		// Parse output for the URL
		for _, line := range strings.Split(outputStr, "\n") {
			if strings.Contains(line, "http://localhost") || strings.Contains(line, "http://127.0.0.1") {
				for _, word := range strings.Fields(line) {
					if strings.HasPrefix(word, "http://") {
						ui.PrintSuccess("Site running at %s", word)
						break
					}
				}
			}
		}

		fmt.Println()
	} else {
		ui.PrintWarning("Skipping preview")
		fmt.Println()
	}

	fmt.Println(ui.Divider())
	ui.PrintSuccess("Warp complete!")
}

// findBackupName returns the next available backup name (e.g. work/site.bak1, .bak2, etc.)
func findBackupName(baseDir string) string {
	for i := 1; ; i++ {
		backup := fmt.Sprintf("%s.bak%d", baseDir, i)
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			return backup
		}
	}
}

// deriveNameFromURL extracts a site name from a URL
func deriveNameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "site"
	}

	hostname := parsed.Hostname()
	// Strip www. prefix
	hostname = strings.TrimPrefix(hostname, "www.")
	// Take first segment before dot
	parts := strings.Split(hostname, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return "site"
}
