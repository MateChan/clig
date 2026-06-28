package server

import (
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/MateChan/clig/internal/models"
)

type Server struct {
	RootDir       string
	CacheManager  *CacheManager
	StaticFS      http.FileSystem
	treeMu        sync.RWMutex
	treeCache     *models.DirNode
	treeCacheTime time.Time
	thumbSem      chan struct{}
	initDone      chan struct{}
}

func NewServer(rootDir string, cm *CacheManager, staticFS http.FileSystem) *Server {
	limit := runtime.NumCPU()
	if limit < 2 {
		limit = 2
	} else if limit > 8 {
		limit = 8
	}

	s := &Server{
		RootDir:      rootDir,
		CacheManager: cm,
		StaticFS:     staticFS,
		thumbSem:     make(chan struct{}, limit),
		initDone:     make(chan struct{}),
	}

	go s.eagerScan()
	return s
}

func (s *Server) StartServer(ln net.Listener) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		s.HandleIndex()(w, r)
	})

	mux.HandleFunc("/api/images", s.HandleImagesAPI())
	mux.HandleFunc("/thumbnail", s.HandleThumbnail())
	mux.HandleFunc("/image", s.HandleOriginal())
	mux.Handle("/static/", http.FileServer(s.StaticFS))

	return http.Serve(ln, mux)
}
