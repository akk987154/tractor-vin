package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seb/tractor-vin/decoder"
)

func do(t *testing.T, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newRouter().ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) DecodeResponse {
	t.Helper()
	var resp DecodeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v（body=%q）", err, w.Body.String())
	}
	return resp
}

// 17 位纯数字会命中 John Deere 的 ^[A-Z0-9]{17}$，作为"必定能解码"的样本
var validSerial = strings.Repeat("1", 17)

func TestHealth(t *testing.T) {
	w := do(t, http.MethodGet, "/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q，期望 application/json 前缀", ct)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q，期望 nosniff", got)
	}
}

func TestBrands(t *testing.T) {
	w := do(t, http.MethodGet, "/brands", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200", w.Code)
	}

	var resp BrandsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v", err)
	}
	if len(resp.Brands) != len(decoder.Brands()) {
		t.Errorf("品牌数 = %d，期望 %d", len(resp.Brands), len(decoder.Brands()))
	}
}

func TestDecodeSuccess(t *testing.T) {
	body, _ := json.Marshal(DecodeRequest{Serial: validSerial})
	w := do(t, http.MethodPost, "/decode", body)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 200（body=%q）", w.Code, w.Body.String())
	}
	resp := decodeBody(t, w)
	if !resp.Success {
		t.Fatalf("success = false，error = %q", resp.Error)
	}
	if resp.Data == nil || resp.Data.Brand == "" {
		t.Fatalf("data 缺失或品牌为空: %+v", resp.Data)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q，期望 application/json 前缀", ct)
	}
}

func TestDecodeRejectsEmptySerial(t *testing.T) {
	for _, serial := range []string{"", "   "} {
		body, _ := json.Marshal(DecodeRequest{Serial: serial})
		w := do(t, http.MethodPost, "/decode", body)

		if w.Code != http.StatusBadRequest {
			t.Errorf("serial=%q 时状态码 = %d，期望 400", serial, w.Code)
		}
		if resp := decodeBody(t, w); resp.Success {
			t.Errorf("serial=%q 时 success 应为 false", serial)
		}
	}
}

func TestDecodeRejectsMalformedJSON(t *testing.T) {
	for _, body := range []string{"", "not json", "{", `{"serial":`} {
		w := do(t, http.MethodPost, "/decode", []byte(body))
		if w.Code != http.StatusBadRequest {
			t.Errorf("body=%q 时状态码 = %d，期望 400", body, w.Code)
		}
	}
}

// 长度上限：请求体本身很小，但 serial 字段超长，应在字段校验处被拦下
func TestDecodeRejectsOverlongSerial(t *testing.T) {
	body, _ := json.Marshal(DecodeRequest{Serial: strings.Repeat("1", decoder.MaxSerialLength+1)})
	w := do(t, http.MethodPost, "/decode", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d，期望 400", w.Code)
	}
	if resp := decodeBody(t, w); resp.Success {
		t.Error("success 应为 false")
	}
}

// 请求体上限：原先直接 json.NewDecoder(r.Body).Decode()，没有任何大小限制，
// 一个超大请求就能把进程内存吃光。超出 maxBodyBytes 应返回 413。
func TestDecodeRejectsHugeBody(t *testing.T) {
	huge := `{"serial":"` + strings.Repeat("A", maxBodyBytes*4) + `"}`
	w := do(t, http.MethodPost, "/decode", []byte(huge))

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("状态码 = %d，期望 413（body 长度 %d）", w.Code, len(huge))
	}
	if resp := decodeBody(t, w); resp.Success {
		t.Error("success 应为 false")
	}
}

// 无法识别的序列号应返回 400，而不是 200。
// CHANGELOG 记录过这个缺陷：解码失败时不调用 WriteHeader，默认返回 200 OK。
func TestDecodeReturnsBadRequestForUnknownSerial(t *testing.T) {
	body, _ := json.Marshal(DecodeRequest{Serial: "1ZZZZ"})
	w := do(t, http.MethodPost, "/decode", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("状态码 = %d，期望 400（body=%q）", w.Code, w.Body.String())
	}
	resp := decodeBody(t, w)
	if resp.Success {
		t.Error("success 应为 false")
	}
	if resp.Error == "" {
		t.Error("应返回错误说明")
	}
}

func TestCORSPreflight(t *testing.T) {
	w := do(t, http.MethodOptions, "/decode", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("状态码 = %d，期望 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q，期望 *", got)
	}
}

// 各路由都必须在有限时间内返回。
// 原实现用 http.ListenAndServe，四个超时全为 0，这里至少验证处理链路不会挂住。
func TestRoutesRespondQuickly(t *testing.T) {
	for _, path := range []string{"/health", "/brands"} {
		w := do(t, http.MethodGet, path, nil)
		if w.Code != http.StatusOK {
			t.Errorf("GET %s 状态码 = %d", path, w.Code)
		}
	}
}
