package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"warp/core/lib/ui"

	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

// Config holds scraper configuration
type Config struct {
	URL       string
	OutputDir string
	Timeout   time.Duration
}

// Result holds scraper output
type Result struct {
	Title     string
	HTML      string
	OutputDir string
	Assets    []string
}

// Scrape navigates to a URL with headless Chrome, captures the rendered HTML and assets
func Scrape(cfg Config) (*Result, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	// Create output directories
	assetsDir := filepath.Join(cfg.OutputDir, "assets")
	for _, dir := range []string{
		cfg.OutputDir,
		filepath.Join(assetsDir, "images"),
		filepath.Join(assetsDir, "css"),
		filepath.Join(assetsDir, "js"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	sp := ui.NewSpinner("Launching browser...")

	// Launch headless Chrome with timeout on the allocator
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer timeoutCancel()

	allocCtx, allocCancel := chromedp.NewExecAllocator(timeoutCtx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-web-security", true),
			chromedp.Flag("disable-extensions", true),
			chromedp.WindowSize(1920, 1080),
		)...,
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var renderedHTML string
	var title string

	sp.Update("Loading page...")
	// Navigate and wait for JS rendering
	err := chromedp.Run(ctx,
		chromedp.Navigate(cfg.URL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			sp.Update("Waiting for JS...")
			return nil
		}),
		chromedp.Sleep(5*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			sp.Update("Capturing HTML...")
			return nil
		}),
		chromedp.Title(&title),
		chromedp.OuterHTML("html", &renderedHTML),
	)
	sp.Finish()
	if err != nil {
		return nil, fmt.Errorf("scraping %s: %w", cfg.URL, err)
	}

	// Parse and extract asset URLs
	assetURLs := extractAssetURLs(renderedHTML, cfg.URL)

	// Download assets and build rewrite map
	rewriteMap := make(map[string]string)
	var downloadedAssets []string

	sp2 := ui.NewSpinner(fmt.Sprintf("Downloading assets (0/%d)...", len(assetURLs)))
	for i, assetURL := range assetURLs {
		sp2.Update(fmt.Sprintf("Downloading assets (%d/%d)...", i+1, len(assetURLs)))
		localPath, err := downloadAsset(assetURL, cfg.URL, assetsDir)
		if err != nil {
			continue // Skip failed downloads
		}
		relPath, _ := filepath.Rel(cfg.OutputDir, localPath)
		rewriteMap[assetURL] = relPath
		downloadedAssets = append(downloadedAssets, localPath)
	}
	sp2.Finish()

	// Rewrite URLs in HTML
	rewrittenHTML := rewriteHTMLURLs(renderedHTML, rewriteMap)

	// Write raw index.html
	indexPath := filepath.Join(cfg.OutputDir, "index.html")
	if err := os.WriteFile(indexPath, []byte(rewrittenHTML), 0644); err != nil {
		return nil, fmt.Errorf("writing index.html: %w", err)
	}

	// Write cleaned version (stripped of scripts, styles, data attributes)
	cleanedHTML := cleanHTML(rewrittenHTML)
	cleanPath := filepath.Join(cfg.OutputDir, "content.html")
	if err := os.WriteFile(cleanPath, []byte(cleanedHTML), 0644); err != nil {
		return nil, fmt.Errorf("writing content.html: %w", err)
	}

	return &Result{
		Title:     title,
		HTML:      rewrittenHTML,
		OutputDir: cfg.OutputDir,
		Assets:    downloadedAssets,
	}, nil
}

// extractAssetURLs parses HTML and extracts image, CSS, and JS URLs
func extractAssetURLs(htmlContent string, baseURL string) []string {
	var urls []string
	seen := make(map[string]bool)

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return urls
	}

	base, _ := url.Parse(baseURL)

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			var attrName string
			switch n.Data {
			case "img":
				attrName = "src"
			case "link":
				attrName = "href"
			case "script":
				attrName = "src"
			}

			if attrName != "" {
				for _, attr := range n.Attr {
					if attr.Key == attrName && attr.Val != "" {
						absURL := resolveURL(attr.Val, base)
						if absURL != "" && !seen[absURL] {
							seen[absURL] = true
							urls = append(urls, absURL)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return urls
}

// resolveURL resolves a potentially relative URL against a base
func resolveURL(rawURL string, base *url.URL) string {
	if rawURL == "" || strings.HasPrefix(rawURL, "data:") {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	resolved := base.ResolveReference(parsed)
	return resolved.String()
}

// downloadAsset downloads a URL to the appropriate assets subdirectory
func downloadAsset(assetURL string, baseURL string, assetsDir string) (string, error) {
	resp, err := http.Get(assetURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, assetURL)
	}

	// Determine subdirectory based on content type or extension
	subdir := categorizeAsset(assetURL, resp.Header.Get("Content-Type"))
	destDir := filepath.Join(assetsDir, subdir)

	// Extract filename from URL
	parsed, _ := url.Parse(assetURL)
	filename := filepath.Base(parsed.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "file"
	}

	destPath := filepath.Join(destDir, filename)

	// Avoid overwriting by appending a number
	if _, err := os.Stat(destPath); err == nil {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		for i := 1; ; i++ {
			destPath = filepath.Join(destDir, fmt.Sprintf("%s_%d%s", name, i, ext))
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
		}
	}

	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return destPath, nil
}

// categorizeAsset determines the asset subdirectory
func categorizeAsset(assetURL string, contentType string) string {
	lower := strings.ToLower(assetURL)

	// Check by extension first
	switch {
	case strings.HasSuffix(lower, ".css"):
		return "css"
	case strings.HasSuffix(lower, ".js"):
		return "js"
	case strings.HasSuffix(lower, ".png"),
		strings.HasSuffix(lower, ".jpg"),
		strings.HasSuffix(lower, ".jpeg"),
		strings.HasSuffix(lower, ".gif"),
		strings.HasSuffix(lower, ".svg"),
		strings.HasSuffix(lower, ".webp"),
		strings.HasSuffix(lower, ".ico"):
		return "images"
	}

	// Fallback to content type
	switch {
	case strings.Contains(contentType, "css"):
		return "css"
	case strings.Contains(contentType, "javascript"):
		return "js"
	case strings.Contains(contentType, "image"):
		return "images"
	}

	return "images" // Default to images
}

// rewriteHTMLURLs replaces absolute asset URLs with local relative paths
func rewriteHTMLURLs(htmlContent string, rewriteMap map[string]string) string {
	result := htmlContent
	for absURL, relPath := range rewriteMap {
		result = strings.ReplaceAll(result, absURL, relPath)
	}
	return result
}

// cleanHTML strips scripts, styles, SVGs, data attributes, and inline styles
// to produce a content-only HTML file that's small enough for Claude to read
func cleanHTML(rawHTML string) string {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return rawHTML
	}

	// Remove unwanted elements
	removeElements(doc)

	// Clean attributes on remaining elements
	cleanAttributes(doc)

	// Render back to string
	var buf strings.Builder
	html.Render(&buf, doc)
	return buf.String()
}

// removeElements removes script, noscript, and iframe elements
func removeElements(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if c.Type == html.ElementNode {
			switch c.Data {
			case "script", "noscript", "iframe":
				n.RemoveChild(c)
				continue
			}
		}
		// Also remove HTML comments
		if c.Type == html.CommentNode {
			n.RemoveChild(c)
			continue
		}
		removeElements(c)
	}
}

// cleanAttributes removes data-* attributes, inline styles, class, and other noise
func cleanAttributes(n *html.Node) {
	if n.Type == html.ElementNode {
		var cleaned []html.Attribute
		for _, attr := range n.Attr {
			// Keep only essential attributes
			switch {
			case attr.Key == "href":
				cleaned = append(cleaned, attr)
			case attr.Key == "src":
				cleaned = append(cleaned, attr)
			case attr.Key == "alt":
				cleaned = append(cleaned, attr)
			case attr.Key == "title":
				cleaned = append(cleaned, attr)
			case attr.Key == "id":
				cleaned = append(cleaned, attr)
			case attr.Key == "lang":
				cleaned = append(cleaned, attr)
			case attr.Key == "charset":
				cleaned = append(cleaned, attr)
			case attr.Key == "name" && n.Data == "meta":
				cleaned = append(cleaned, attr)
			case attr.Key == "content" && n.Data == "meta":
				cleaned = append(cleaned, attr)
			case attr.Key == "rel":
				cleaned = append(cleaned, attr)
			case attr.Key == "type":
				cleaned = append(cleaned, attr)
			case attr.Key == "target":
				cleaned = append(cleaned, attr)
			}
		}
		n.Attr = cleaned
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		cleanAttributes(c)
	}
}
