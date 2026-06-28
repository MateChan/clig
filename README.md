# CLI Gallery

A simple command-line tool to display images in a specified directory as a
gallery in your browser.

## Installation

1. Download the binary from GitHub Actions Artifacts.
1. Place it in a directory included in your PATH.

## Usage

```
Usage: clig [options] <target_directory>

Options:
  -cache-dir string
        Directory to store generated thumbnails (defaults to user cache directory)
  -host string
        Host interface to bind to (default "0.0.0.0")
  -port int
        Port to run the server on (default 9411)
```

## Build from source

```bash
git clone https://github.com/MateChan/clig.git
cd clig
go tool templ generate
go build -o clig
```

### Hot Reload

```bash
go tool air
```
