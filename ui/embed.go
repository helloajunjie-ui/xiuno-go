package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var staticFiles embed.FS

// GetStaticFS 返回嵌入的静态文件系统（剥离 dist/ 前缀）
func GetStaticFS() http.FileSystem {
	fsys, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}
