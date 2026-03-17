package cmd

import (
	"fmt"
	"os"

	"warp/core/lib/ui"

	"github.com/spf13/cobra"
)

// Version is set via ldflags during build
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "warp [url]",
	Short: "Warp — modernize any website with AI",
	Long: ui.Banner() + "\n" +
		ui.VersionLine(Version) + "\n\n" +
		ui.Divider() + "\n\n" +
		"  Warp takes an existing website, scrapes it, modernizes the design\n" +
		"  using Claude, and deploys it via lightspeed.\n\n" +
		"  Usage: warp <url>",
	Args: cobra.MaximumNArgs(1),
	Run:  runWarp,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}

func init() {
	rootCmd.Flags().StringVarP(&warpName, "name", "n", "", "Site name (defaults to hostname)")
	rootCmd.Flags().StringVarP(&warpDir, "dir", "d", "work", "Working directory")
	rootCmd.Flags().BoolVar(&warpSkipScrape, "skip-scrape", false, "Skip the scrape phase (reuse existing)")
	rootCmd.Flags().BoolVar(&warpSkipDeploy, "skip-deploy", false, "Skip the deploy phase")
	rootCmd.Flags().BoolVar(&warpSkipModern, "skip-modernize", false, "Skip the modernize phase")
	rootCmd.Flags().IntVarP(&warpTimeout, "timeout", "t", 120, "Scraping timeout in seconds")
	rootCmd.Flags().StringVar(&warpURL, "url", "", "URL to warp (alternative to positional arg)")
	rootCmd.Flags().StringVarP(&warpInstructions, "instructions", "i", "", "Custom instructions for Claude (e.g. colors, style, layout)")

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.PrintError("%s", err)
		os.Exit(1)
	}
}
