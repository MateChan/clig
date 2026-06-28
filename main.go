package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MateChan/clig/internal/models"
	"github.com/MateChan/clig/internal/server"
	"github.com/MateChan/clig/internal/web"
)

func main() {
	host := flag.String("host", "0.0.0.0", "Host interface to bind to")
	port := flag.Int("port", 9411, "Port to run the server on")
	cacheDir := flag.String("cache-dir", "", "Directory to store generated thumbnails (defaults to user cache directory)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <target_directory>\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: target_directory is required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	targetDir := flag.Arg(0)

	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		log.Fatalf("Error resolving path: %v", err)
	}

	info, err := os.Stat(absTargetDir)
	if err != nil {
		log.Fatalf("Error reading target directory: %v", err)
	}
	if !info.IsDir() {
		log.Fatalf("Error: target_directory must be a directory: %s", absTargetDir)
	}

	cm, err := server.NewCacheManager(*cacheDir)
	if err != nil {
		log.Fatalf("Failed to initialize thumbnail cache: %v", err)
	}

	srv := server.NewServer(absTargetDir, cm, http.FS(web.StaticFS))

	var ln net.Listener
	var listenPort int

	portSpecified := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portSpecified = true
		}
	})

	if portSpecified {
		addr := fmt.Sprintf("%s:%d", *host, *port)
		var err error
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			log.Fatalf("Failed to bind to port %d: %v", *port, err)
		}
		listenPort = *port
	} else {
		const maxTries = 100
		var err error

		for p := *port; p < *port+maxTries; p++ {
			addr := fmt.Sprintf("%s:%d", *host, p)
			ln, err = net.Listen("tcp", addr)
			if err == nil {
				listenPort = p
				break
			}
		}

		if listenPort == 0 {
			log.Fatalf("Failed to find an available port starting from %d after %d retries: %v", *port, maxTries, err)
		}
	}

	log.Printf("Starting %s Server...", models.APP_NAME)
	log.Printf("Serving Directory: %s", absTargetDir)
	log.Printf("Cache Directory:   %s", cm.CacheDir)

	if *host == "0.0.0.0" {
		log.Printf("Server URL:        http://localhost:%d", listenPort)
		if ips, err := getLocalIPs(); err == nil {
			for _, ip := range ips {
				log.Printf("Server URL (LAN):  http://%s:%d", ip, listenPort)
			}
		}
	} else {
		log.Printf("Server URL:        http://%s:%d", *host, listenPort)
	}

	err = srv.StartServer(ln)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getLocalIPs() ([]string, error) {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if (iface.Flags&net.FlagUp) == 0 || (iface.Flags&net.FlagLoopback) != 0 {
			continue
		}

		name := iface.Name
		isVirtual := false
		virtualKeywords := []string{"veth", "docker", "br-", "virbr", "vmnet", "vboxnet"}
		for _, keyword := range virtualKeywords {
			if strings.Contains(strings.ToLower(name), keyword) {
				isVirtual = true
				break
			}
		}
		if isVirtual {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && !ipNet.IP.IsLoopback() {
				if ipv4 := ipNet.IP.To4(); ipv4 != nil {
					ips = append(ips, ipv4.String())
				}
			}
		}
	}
	return ips, nil
}
