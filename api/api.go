package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/seb/tractor-vin/decoder"
)

// maxBodyBytes 限制请求体大小。
// 原先 json.NewDecoder(r.Body).Decode(&req) 直接读裸 body，
// 没有任何上限，一个超大请求就能把进程内存吃光。
const maxBodyBytes = 4 << 10 // 4 KiB

type DecodeRequest struct {
	Serial string `json:"serial"`
}

type DecodeResponse struct {
	Success bool                 `json:"success"`
	Data    *decoder.DecodedInfo `json:"data,omitempty"`
	Error   string               `json:"error,omitempty"`
}

type BrandsResponse struct {
	Brands []string `json:"brands"`
}

// writeJSON 统一写响应：设置 Content-Type 与 nosniff。
// 原实现直接 json.NewEncoder(w).Encode(...)，Go 会嗅探成 text/plain，
// 且所有 Encode 错误被丢弃。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("写出响应失败: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, DecodeResponse{Success: false, Error: msg})
}

// newRouter 组装路由与中间件。
// 单独抽出来是为了让测试可以直接用 httptest 打这套路由，
// 而不必真的去监听端口。
func newRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/brands", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, BrandsResponse{Brands: decoder.Brands()})
	})

	r.Post("/decode", func(w http.ResponseWriter, r *http.Request) {
		// 逐层限制输入规模：body 大小 → 字段长度 → 解码器内部
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

		var req DecodeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "请求体过大")
				return
			}
			writeError(w, http.StatusBadRequest, "无效的请求格式")
			return
		}

		serial := strings.TrimSpace(req.Serial)
		if serial == "" {
			writeError(w, http.StatusBadRequest, "序列号不能为空")
			return
		}
		if len(serial) > decoder.MaxSerialLength {
			writeError(w, http.StatusBadRequest,
				fmt.Sprintf("序列号长度不能超过 %d 个字符", decoder.MaxSerialLength))
			return
		}

		result, err := decoder.Decode(serial)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, DecodeResponse{Success: true, Data: result})
	})

	return r
}

// Serve 启动 HTTP 服务，返回错误而不是直接 os.Exit，
// 以便调用方决定如何处理（也便于测试与优雅退出）。
func Serve(host string, port int) error {
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: newRouter(),
		// 原实现用 http.ListenAndServe，四个超时全部为 0（永不超时），
		// 一个慢速客户端就能长期占住连接（Slowloris）。
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("TractorVIN API 监听 http://%s", srv.Addr)
	return srv.ListenAndServe()
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
