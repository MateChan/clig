package server

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/MateChan/clig/internal/models"
	"github.com/MateChan/clig/internal/web/templates"
)

const PAGE_SIZE = 24

func (s *Server) HandleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forceRefresh := r.URL.Query().Get("refresh") == "true"

		if forceRefresh {
			tree, err := ScanDir(s.RootDir)
			if err != nil {
				http.Error(w, "Failed to scan directory: "+err.Error(), http.StatusInternalServerError)
				return
			}

			s.treeMu.Lock()
			s.treeCache = tree
			s.treeCacheTime = time.Now()
			s.treeMu.Unlock()

			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		var tree *models.DirNode
		var err error

		s.treeMu.RLock()
		if s.treeCache != nil {
			tree = s.treeCache
			s.treeMu.RUnlock()
		} else {
			s.treeMu.RUnlock()
			<-s.initDone

			s.treeMu.RLock()
			tree = s.treeCache
			s.treeMu.RUnlock()

			if tree == nil {
				tree, err = ScanDir(s.RootDir)
				if err != nil {
					http.Error(w, "Failed to scan directory: "+err.Error(), http.StatusInternalServerError)
					return
				}

				s.treeMu.Lock()
				s.treeCache = tree
				s.treeCacheTime = time.Now()
				s.treeMu.Unlock()
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := templates.Page(tree, s.RootDir).Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	}
}

func (s *Server) HandleImagesAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Query().Get("hash")
		pageStr := r.URL.Query().Get("page")

		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}

		var targetDir string
		if hash == "" {
			targetDir = s.RootDir
		} else {
			targetDir, err = ResolveSafePath(s.RootDir, hash)
			if err != nil {
				http.Error(w, "Unauthorized or invalid directory path", http.StatusForbidden)
				return
			}
		}

		images, err := ListImageEntries(s.RootDir, targetDir[len(s.RootDir):])
		if err != nil {
			http.Error(w, "Failed to read directory: "+err.Error(), http.StatusInternalServerError)
			return
		}

		totalImages := len(images)
		offset := (page - 1) * PAGE_SIZE

		var paginatedImages []models.ImageInfo
		hasMore := false

		if offset < totalImages {
			end := offset + PAGE_SIZE
			if end > totalImages {
				end = totalImages
			}

			paginatedImages = images[offset:end]
			hasMore = end < totalImages
		}

		paginatedImages = LoadImagesMetadata(s.RootDir, paginatedImages)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Image-Count", strconv.Itoa(totalImages))
		w.Header().Set("X-Has-More", strconv.FormatBool(hasMore))

		if err := templates.ImageCards(paginatedImages).Render(r.Context(), w); err != nil {
			log.Printf("Error rendering template: %v", err)
		}
	}
}

func (s *Server) HandleThumbnail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Query().Get("hash")
		if hash == "" {
			http.Error(w, "Missing hash parameter", http.StatusBadRequest)
			return
		}

		filePath, err := ResolveSafePath(s.RootDir, hash)
		if err != nil {
			http.Error(w, "Unauthorized or invalid image path", http.StatusForbidden)
			return
		}

		s.thumbSem <- struct{}{}
		thumbPath, err := s.CacheManager.GetThumbnailPath(filePath)
		<-s.thumbSem
		if err != nil {
			http.Error(w, "Failed to generate thumbnail: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=604800") // 1 week
		http.ServeFile(w, r, thumbPath)
	}
}

func (s *Server) HandleOriginal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash := r.URL.Query().Get("hash")
		if hash == "" {
			http.Error(w, "Missing hash parameter", http.StatusBadRequest)
			return
		}

		filePath, err := ResolveSafePath(s.RootDir, hash)
		if err != nil {
			http.Error(w, "Unauthorized or invalid image path", http.StatusForbidden)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=86400") // 1 day
		http.ServeFile(w, r, filePath)
	}
}

func (s *Server) eagerScan() {
	defer close(s.initDone)

	log.Printf("Pre-scanning tree in background: %s", s.RootDir)
	start := time.Now()
	tree, err := ScanDir(s.RootDir)
	if err != nil {
		log.Printf("Warning: Pre-scan failed: %v", err)
		return
	}

	s.treeMu.Lock()
	s.treeCache = tree
	s.treeCacheTime = time.Now()
	s.treeMu.Unlock()

	log.Printf("Pre-scan completed in %v", time.Since(start))
}
