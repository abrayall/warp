package modernizer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"warp/core/lib/ui"
)

// Config holds modernizer configuration
type Config struct {
	Name         string
	BaseDir      string // Parent dir containing scraped/ and site/
	ScrapedDir   string
	OutputDir    string
	Instructions string // Custom user instructions for Claude
}

// Result holds modernizer output
type Result struct {
	SiteDir string
}

// Modernize takes scraped HTML and uses Claude to create a modern lightspeed PHP site
func Modernize(cfg Config) (*Result, error) {
	// Create lightspeed project structure
	if err := createProjectStructure(cfg); err != nil {
		return nil, fmt.Errorf("creating project structure: %w", err)
	}

	// Copy images from scraped assets to site assets
	if err := copyImages(cfg); err != nil {
		// Non-fatal — images might not exist
		_ = err
	}

	// Build Claude prompt using relative paths
	prompt := buildPrompt(cfg.Name, cfg.Instructions)

	// Run Claude from the base dir so both scraped/ and site/ are accessible
	logPath := filepath.Join(cfg.BaseDir, "warp.log")
	if err := runClaude(prompt, cfg.BaseDir, logPath); err != nil {
		return nil, fmt.Errorf("running Claude: %w", err)
	}

	return &Result{
		SiteDir: cfg.OutputDir,
	}, nil
}

// createProjectStructure sets up the lightspeed project directories and site.properties
func createProjectStructure(cfg Config) error {
	dirs := []string{
		cfg.OutputDir,
		filepath.Join(cfg.OutputDir, "assets"),
		filepath.Join(cfg.OutputDir, "assets", "images"),
		filepath.Join(cfg.OutputDir, "assets", "css"),
		filepath.Join(cfg.OutputDir, "assets", "js"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Write site.properties
	props := fmt.Sprintf("name=%s\n", cfg.Name)
	propsPath := filepath.Join(cfg.OutputDir, "site.properties")
	if err := os.WriteFile(propsPath, []byte(props), 0644); err != nil {
		return err
	}

	return nil
}

// copyImages copies image assets from scraped directory to site directory
func copyImages(cfg Config) error {
	srcDir := filepath.Join(cfg.ScrapedDir, "assets", "images")
	dstDir := filepath.Join(cfg.OutputDir, "assets", "images")

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())

		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			continue
		}
	}

	return nil
}

// buildPrompt creates the Claude prompt for modernizing the site
func buildPrompt(name string, instructions string) string {
	prompt := fmt.Sprintf(`You are modernizing a website called "%s".

The original website has been scraped and saved to the scraped/ directory.
- Read scraped/content.html to understand the site content and structure (this is a cleaned version with scripts/styles removed)
- The scraped assets (images, css, js) are in scraped/assets/

Your job is to create a modern, redesigned version of this website as a lightspeed site in the site/ directory.

## What is lightspeed?

Lightspeed is a simple PHP hosting platform. A lightspeed site is just a directory with:
- site.properties (contains name=<sitename> — already created for you)
- index.php (the main page)
- assets/css/ (stylesheets)
- assets/js/ (javascript)
- assets/images/ (images — already copied from the scraped site)

You deploy a lightspeed site by running "lightspeed deploy" in the site directory.

## Your working directory

Your working directory contains:
- scraped/ — the scraped original site (read from here)
- site/ — the lightspeed project to build (write files here)

The site/ directory already has the lightspeed structure set up with site.properties and the images copied.

## Instructions

1. Read scraped/content.html to understand the content on the site
2. Create site/index.php using the original content as a base, but IMPROVE it:
   - Fix any spelling and grammar errors
   - Simplify overly complex or wordy sentences — make them clearer and easier to read
   - Optimize for SEO (proper heading hierarchy, meta descriptions, keyword usage)
   - Improve readability and flow
   - Keep the same topics, services, and messaging — just make it better
3. Create site/assets/css/style.css with clean, hand-crafted CSS
4. The design should be modern, clean, and professional
5. Must be fully responsive (mobile-first)
6. Use semantic HTML5 elements
7. Add smooth, subtle animations and transitions
8. Use a clean color palette inspired by the original site
9. Reference images from assets/images/
10. Navigation header behavior:
    - The header/nav bar should be TRANSPARENT when the page is at the top, so the hero image shows through
    - When the user scrolls down, the header should get a solid background color (with a smooth transition)
    - Use fixed/sticky positioning so the nav stays at the top while scrolling
11. Active section highlighting:
    - As the user scrolls, highlight the current section's link in the navigation
    - Use JavaScript with IntersectionObserver or scroll events to detect which section is in view
    - Only do this if the site has distinct sections that map to nav links
12. Scroll to top button:
    - Add a "scroll to top" button that appears when the user scrolls down
    - It should smoothly scroll back to the top when clicked
    - Hide it when the user is already at the top
13. Use a small amount of vanilla JavaScript (inline or in assets/js/main.js) for the scroll behaviors above

IMPORTANT:
- Do NOT use any CSS frameworks (no Bootstrap, Tailwind, etc). Write all CSS by hand.
- Do NOT invent new services, claims, or information that weren't on the original site
- Do NOT remove sections or topics from the original site
- Do NOT modify site.properties
- Keep it simple — index.php, assets/css/style.css, and optionally assets/js/main.js`, name)

	if instructions != "" {
		prompt += fmt.Sprintf(`


## Additional instructions from the user

%s`, instructions)
	}

	return prompt
}

// streamEvent represents a parsed JSON event from claude stream-json output
type streamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Message struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			Name  string `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	} `json:"message"`
	ToolUseResult interface{} `json:"tool_use_result"`
}

// toolInput is used to extract file_path from tool inputs
type toolInput struct {
	FilePath string `json:"file_path"`
	Command  string `json:"command"`
	Content  string `json:"content"`
}

// runClaude invokes the Claude CLI to generate the site files
func runClaude(prompt string, workDir string, logPath string) error {
	// Open log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}
	defer logFile.Close()

	logf := func(format string, a ...interface{}) {
		msg := fmt.Sprintf(format, a...)
		fmt.Fprintf(logFile, "[%s] %s\n", time.Now().Format("15:04:05"), msg)
	}

	// Log the full prompt
	logf("=== WARP MODERNIZE SESSION ===")
	logf("Working directory: %s", workDir)
	logf("")
	logf("=== PROMPT ===")
	fmt.Fprintln(logFile, prompt)
	logf("")
	logf("=== CLAUDE SESSION ===")

	sp := ui.NewSpinner("Starting Claude...")

	cmd := exec.Command("claude", "-p",
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
	)
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stderr = os.Stderr

	logf("Command: claude -p --output-format stream-json --verbose --dangerously-skip-permissions")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sp.Finish()
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		sp.Finish()
		if strings.Contains(err.Error(), "executable file not found") {
			return fmt.Errorf("claude CLI not found — install it from https://claude.ai/code")
		}
		return fmt.Errorf("starting claude: %w", err)
	}

	logf("Claude started (PID %d)", cmd.Process.Pid)

	// Parse stream-json events and show progress
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large lines

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Log raw JSON event
		fmt.Fprintln(logFile, line)

		var event streamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		switch event.Type {
		case "system":
			if event.Subtype == "init" {
				logf("[STATUS] Session initialized")
				sp.Update("Claude is thinking...")
			}

		case "assistant":
			for _, content := range event.Message.Content {
				switch content.Type {
				case "text":
					text := strings.TrimSpace(content.Text)
					if text != "" {
						logf("[CLAUDE] %s", text)
					}

				case "thinking":
					logf("[THINKING] (extended thinking block)")

				case "tool_use":
					var input toolInput
					json.Unmarshal(content.Input, &input)

					logf("[TOOL] %s", content.Name)
					if input.FilePath != "" {
						logf("  file: %s", input.FilePath)
					}
					if input.Command != "" {
						logf("  command: %s", input.Command)
					}

					switch content.Name {
					case "Read":
						file := filepath.Base(input.FilePath)
						sp.Update(fmt.Sprintf("Reading %s", file))
					case "Write":
						file := filepath.Base(input.FilePath)
						sp.Update(fmt.Sprintf("Writing %s", file))
					case "Edit":
						file := filepath.Base(input.FilePath)
						sp.Update(fmt.Sprintf("Editing %s", file))
					case "Bash":
						desc := input.Command
						if idx := strings.IndexAny(desc, "\n\r"); idx != -1 {
							desc = desc[:idx]
						}
						desc = strings.TrimSpace(desc)
						if len(desc) > 50 {
							desc = desc[:50] + "..."
						}
						if desc == "" {
							desc = "command"
						}
						sp.Update(fmt.Sprintf("Running: %s", desc))
					case "Glob":
						sp.Update("Searching files...")
					case "Grep":
						sp.Update("Searching content...")
					default:
						sp.Update(fmt.Sprintf("Using %s...", content.Name))
					}
				}
			}

		case "user":
			// Tool results
			logf("[RESULT] Tool result received")
		}
	}

	cmdErr := cmd.Wait()
	sp.Finish()

	if cmdErr != nil {
		logf("[ERROR] Claude exited with error: %s", cmdErr)
		return fmt.Errorf("claude exited with error: %w", cmdErr)
	}

	logf("[DONE] Claude finished successfully")
	return nil
}

// ListImages returns a list of image filenames in the site's assets/images directory
func ListImages(siteDir string) []string {
	var images []string
	imagesDir := filepath.Join(siteDir, "assets", "images")

	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return images
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			images = append(images, entry.Name())
		}
	}

	return images
}
