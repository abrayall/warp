# Warp - Website Modernization Tool

Warp takes any existing website, scrapes it, modernizes the design and content using AI, and deploys it via [lightspeed](https://github.com/abrayall/lightspeed).

```
 █   █ █▀▀█ █▀▀█ █▀▀█
 █▄█▄█ █▄▄█ █▄▄▀ █▄▄▀
  ▀ ▀  ▀  ▀ ▀  ▀ ▀
```

## How it works

Warp runs a three-phase pipeline:

1. **Scrape** — Launches headless Chrome via [chromedp](https://github.com/chromedp/chromedp), waits for JavaScript to render, captures the full DOM, downloads assets (images, CSS, JS), and produces a cleaned `content.html` stripped of scripts and noise
2. **Modernize** — Sends the scraped content to [Claude Code](https://claude.ai/code) which reads the original site, improves the copy (spelling, grammar, SEO, readability), and builds a modern responsive PHP site with hand-crafted CSS
3. **Preview** — Starts a local development server via `lightspeed start` so you can see the result immediately

## Install

```bash
curl -sfL https://raw.githubusercontent.com/abrayall/warp/main/install.sh | sh
```

Or build from source:

```bash
git clone git@github.com:abrayall/warp.git
cd warp
./install.sh
```

## Usage

```bash
# Full pipeline: scrape → modernize → preview
warp https://example.com

# Skip scraping (reuse previous scrape)
warp https://example.com --skip-scrape

# Skip modernization (just scrape)
warp https://example.com --skip-modernize --skip-deploy

# Custom site name and working directory
warp https://example.com --name mysite --dir output

# Increase scraping timeout for slow sites
warp https://example.com --timeout 180

# Custom instructions for the AI modernization
warp https://example.com -i "Use navy blue and gold colors. Add a parallax hero section."

# Iterate on modernization without re-scraping
warp https://example.com --skip-scrape -i "Make the fonts larger and add more whitespace"
```

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--name` | `-n` | hostname | Site name |
| `--dir` | `-d` | `work` | Working directory |
| `--timeout` | `-t` | `120` | Scraping timeout in seconds |
| `--instructions` | `-i` | | Custom instructions for Claude (colors, style, layout, etc.) |
| `--skip-scrape` | | `false` | Skip scrape phase (reuse existing) |
| `--skip-modernize` | | `false` | Skip modernize phase |
| `--skip-deploy` | | `false` | Skip preview phase |
| `--url` | | | URL (alternative to positional arg) |

## Working directory

Warp organizes its work under `work/<name>/`:

```
work/
└── example/
    ├── scraped/              ← Phase 1 output
    │   ├── index.html           (raw rendered DOM)
    │   ├── content.html         (cleaned for AI consumption)
    │   └── assets/
    │       ├── images/
    │       ├── css/
    │       └── js/
    ├── site/                 ← Phase 2 output (lightspeed project)
    │   ├── site.properties
    │   ├── index.php
    │   └── assets/
    │       ├── images/
    │       ├── css/style.css
    │       └── js/main.js
    └── warp.log              ← Detailed session log
```

Previous runs are automatically backed up to `work/<name>.bak1`, `.bak2`, etc.

## Requirements

- **Go 1.21+**
- **Google Chrome** — for headless scraping
- **Claude Code** — for AI modernization (`claude` CLI on PATH)
- **Lightspeed** — for local preview and deployment (`lightspeed` CLI on PATH)

## Project structure

```
warp/
├── core/lib/
│   ├── ui/            Terminal UI (lipgloss)
│   ├── scraper/       Headless Chrome scraping (chromedp)
│   ├── modernizer/    Claude Code invocation + stream parsing
│   └── publisher/     Lightspeed deploy integration
├── framework/
│   ├── cli/           Cobra CLI entry point
│   └── server/        Server (stub)
├── build.sh           Build script (uses vermouth for versioning)
└── install.sh         Install script
```

## License

MIT
