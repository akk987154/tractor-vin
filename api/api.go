package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/seb/tractor-vin/decoder"
)

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

func Serve(port int) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Get("/brands", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(BrandsResponse{Brands: decoder.Brands()})
	})

	r.Post("/decode", func(w http.ResponseWriter, r *http.Request) {
		var req DecodeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(DecodeResponse{Success: false, Error: "无效的请求格式"})
			return
		}

		if req.Serial == "" {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(DecodeResponse{Success: false, Error: "序列号不能为空"})
			return
		}

		result, err := decoder.Decode(req.Serial)
		if err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(DecodeResponse{Success: false, Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(DecodeResponse{Success: true, Data: result})
	})

	addr := fmt.Sprintf(":%d", port)
	log.Fatal(http.ListenAndServe(addr, r))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		next.ServeHTTP(w, r)
	})
}
