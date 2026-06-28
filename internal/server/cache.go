package server

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

const CACHE_SUBDIR = "cli-gallery-cache"

type CacheManager struct {
	CacheDir string
}

func NewCacheManager(customCacheDir string) (*CacheManager, error) {
	dir := customCacheDir
	if dir == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			cacheDir = os.TempDir()
		}
		dir = filepath.Join(cacheDir, CACHE_SUBDIR)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &CacheManager{CacheDir: dir}, nil
}

func (cm *CacheManager) GetThumbnailPath(originalPath string) (string, error) {
	fileInfo, err := os.Stat(originalPath)
	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(originalPath)
	if err != nil {
		absPath = originalPath
	}

	keySource := fmt.Sprintf("%s|%d|%d", absPath, fileInfo.Size(), fileInfo.ModTime().UnixNano())
	hash := sha256.Sum256([]byte(keySource))

	cacheKey := fmt.Sprintf("%x.jpg", hash)
	cachePath := filepath.Join(cm.CacheDir, cacheKey)

	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	src, err := imaging.Open(originalPath)
	if err != nil {
		return "", err
	}

	dst := imaging.Fit(src, 360, 240, imaging.CatmullRom)

	tmpFile, err := os.CreateTemp(cm.CacheDir, "thumb-*.tmp")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := imaging.Encode(tmpFile, dst, imaging.JPEG); err != nil {
		return "", err
	}

	tmpFile.Close()

	if err := os.Rename(tmpFile.Name(), cachePath); err != nil {
		return "", err
	}

	return cachePath, nil
}
