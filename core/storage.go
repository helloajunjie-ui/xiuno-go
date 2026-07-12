// xiuno-go v2.1.0-beta 尼克修改版
package core

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Storage 存储驱动接口
// 抽象文件存储后端，当前实现 LocalStorage（本地磁盘）
// 未来可扩展 OSSStorage、S3Storage 等，业务层无需修改
type Storage interface {
	// Put 保存文件，返回相对路径（如 202601/02/150405.123456789.jpg）和错误
	Put(reader io.Reader, ext string) (string, error)
	// GetURL 获取外部可访问的完整 URL
	GetURL(path string) string
	// PutFixedPath 保存文件到指定相对路径（覆盖写入，用于头像等固定路径文件）
	PutFixedPath(reader io.Reader, relPath string) error
	// Delete 删除存储中的文件
	Delete(relPath string) error
	// ServeDownload 以附件下载方式响应文件（设置 Content-Disposition 头）
	ServeDownload(w http.ResponseWriter, r *http.Request, relPath, orgFilename string)
}

// LocalStorage 本地文件系统驱动
type LocalStorage struct {
	BaseDir string // 物理存储根目录，如 "upload/"
	BaseURL string // URL 前缀，如 "/upload/"
}

// NewLocalStorage 创建本地存储驱动
func NewLocalStorage(baseDir, baseURL string) *LocalStorage {
	return &LocalStorage{BaseDir: baseDir, BaseURL: baseURL}
}

// Put 保存文件到本地磁盘
// 目录按 YYYYMM/DD/ 散列，文件名使用纳秒时间戳 + 随机后缀
func (s *LocalStorage) Put(reader io.Reader, ext string) (string, error) {
	now := time.Now()
	// 按年月/日打散目录: 202601/02
	subDir := now.Format("200601/02")
	// 生成随机文件名（纳秒时间戳，避免冲突）
	fileName := now.Format("150405.000000000") + ext
	relPath := filepath.Join(subDir, fileName)
	fullPath := filepath.Join(s.BaseDir, relPath)

	// 自动创建嵌套目录
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}

	out, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	if err != nil {
		return "", err
	}

	// 保证返回正斜杠路径（Windows 兼容）
	return filepath.ToSlash(relPath), nil
}

// Delete 删除本地文件
func (s *LocalStorage) Delete(relPath string) error {
	fullPath := filepath.Join(s.BaseDir, relPath)
	return os.Remove(fullPath)
}

// ServeDownload 以附件下载方式响应文件
func (s *LocalStorage) ServeDownload(w http.ResponseWriter, r *http.Request, relPath, orgFilename string) {
	w.Header().Set("Content-Disposition", `attachment; filename="`+orgFilename+`"`)
	fullPath := filepath.Join(s.BaseDir, relPath)
	http.ServeFile(w, r, fullPath)
}

// GetURL 拼接外部可访问的完整 URL
func (s *LocalStorage) GetURL(path string) string {
	return s.BaseURL + path
}

// PutFixedPath 保存文件到指定相对路径（覆盖写入）
// 用于头像等需要固定路径的场景，自动创建父目录
func (s *LocalStorage) PutFixedPath(reader io.Reader, relPath string) error {
	fullPath := filepath.Join(s.BaseDir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	out, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, reader)
	return err
}
