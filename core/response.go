// xiuno-go v2.1.0-beta 尼克修改版
package core

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
)

// Response 统一 API 响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON 写入 JSON 响应
func JSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// JSONSuccess 成功响应
func JSONSuccess(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// JSONError 错误响应
func JSONError(w http.ResponseWriter, httpCode int, msg string) {
	JSON(w, httpCode, Response{
		Code:    -1,
		Message: msg,
	})
}

// JSONErrorLog 错误响应 + 日志记录
// 在返回错误的同时打印详细日志，用于排查 500 类问题
func JSONErrorLog(w http.ResponseWriter, httpCode int, msg string, err error) {
	if err != nil {
		log.Printf("[ERROR] HTTP %d %s | err=%v | stack:\n%s", httpCode, msg, err, debug.Stack())
	} else {
		log.Printf("[ERROR] HTTP %d %s | stack:\n%s", httpCode, msg, debug.Stack())
	}
	JSON(w, httpCode, Response{
		Code:    -1,
		Message: msg,
	})
}
