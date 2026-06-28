package models

import "time"

const APP_NAME = "CLI Gallery"

type DirNode struct {
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Hash      string     `json:"hash"`
	Children  []*DirNode `json:"children"`
	HasImages bool       `json:"hasImages"`
}

type ImageInfo struct {
	Name       string    `json:"name"`
	RelPath    string    `json:"relPath"`
	Hash       string    `json:"hash"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
}
