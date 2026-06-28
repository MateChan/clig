package server

import (
	"encoding/base64"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/MateChan/clig/internal/models"
)

const MAX_WORKERS = 8

func PathToHash(relPath string) string {
	return base64.URLEncoding.EncodeToString([]byte(relPath))
}

func HashToPath(hash string) (string, error) {
	bytes, err := base64.URLEncoding.DecodeString(hash)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ResolveSafePath(rootDir, hash string) (string, error) {
	rel, err := HashToPath(hash)
	if err != nil {
		return "", err
	}

	rel = strings.TrimPrefix(rel, "/")
	rel = strings.TrimPrefix(rel, "\\")
	relClean := filepath.Clean(rel)

	if strings.HasPrefix(relClean, "..") || filepath.IsAbs(relClean) {
		return "", os.ErrPermission
	}

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return "", err
	}

	absTarget := filepath.Join(absRoot, relClean)
	absTargetClean, err := filepath.Abs(absTarget)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absTargetClean, absRoot) {
		return "", os.ErrPermission
	}

	return absTargetClean, nil
}

func IsImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
		return true
	}
	return false
}

func ScanDir(rootDir string) (*models.DirNode, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
	return scanDirRecursive(absRoot, "")
}

func scanDirRecursive(absRoot, relPath string) (*models.DirNode, error) {
	fullPath := filepath.Join(absRoot, relPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(fullPath)
	if relPath == "" {
		name = filepath.Base(absRoot)
		if name == "." || name == "/" || name == "" {
			name = "Root"
		}
	}

	node := &models.DirNode{
		Name:     name,
		Path:     relPath,
		Hash:     PathToHash(relPath),
		Children: []*models.DirNode{},
	}

	hasImages := false
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			childRel := filepath.Join(relPath, name)
			childNode, err := scanDirRecursive(absRoot, childRel)
			if err == nil && childNode != nil {
				if childNode.HasImages || len(childNode.Children) > 0 {
					node.Children = append(node.Children, childNode)
				}
			}
		} else if IsImageFile(entry.Name()) {
			hasImages = true
		}
	}

	node.HasImages = hasImages
	return node, nil
}

func ListImageEntries(rootDir, relPath string) ([]models.ImageInfo, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(absRoot, relPath)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var images []models.ImageInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if IsImageFile(entry.Name()) {
			imgRelPath := filepath.Join(relPath, entry.Name())
			fileInfo, errStat := entry.Info()

			if errStat != nil {
				continue
			}

			images = append(images, models.ImageInfo{
				Name:       entry.Name(),
				RelPath:    imgRelPath,
				Hash:       PathToHash(imgRelPath),
				Size:       fileInfo.Size(),
				ModifiedAt: fileInfo.ModTime(),
			})
		}
	}
	return images, nil
}

func LoadImagesMetadata(rootDir string, images []models.ImageInfo) []models.ImageInfo {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return images
	}

	numImages := len(images)
	if numImages == 0 {
		return images
	}

	numWorkers := MAX_WORKERS
	if numImages < numWorkers {
		numWorkers = numImages
	}

	type result struct {
		index int
		info  models.ImageInfo
		err   error
	}

	jobs := make(chan int, numImages)
	results := make(chan result, numImages)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Go(func() {
			for index := range jobs {
				imgFullPath := filepath.Join(absRoot, images[index].RelPath)
				info, err := GetImageInfo(imgFullPath, images[index].RelPath)
				results <- result{index: index, info: info, err: err}
			}
		})
	}

	for i := range numImages {
		jobs <- i
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for res := range results {
		i := res.index
		if res.err == nil {
			images[i].Width = res.info.Width
			images[i].Height = res.info.Height
			images[i].ModifiedAt = res.info.ModifiedAt
		} else if images[i].Width == 0 || images[i].Height == 0 {
			images[i].Width = 1200
			images[i].Height = 800
		}
	}

	return images
}

func GetImageInfo(fullPath, relPath string) (models.ImageInfo, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return models.ImageInfo{}, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return models.ImageInfo{}, err
	}

	info := models.ImageInfo{
		Name:       filepath.Base(fullPath),
		RelPath:    relPath,
		Hash:       PathToHash(relPath),
		Size:       fileInfo.Size(),
		ModifiedAt: fileInfo.ModTime(),
	}

	config, _, err := image.DecodeConfig(file)
	if err == nil {
		info.Width = config.Width
		info.Height = config.Height
	} else {
		info.Width = 800
		info.Height = 600
	}

	return info, nil
}
