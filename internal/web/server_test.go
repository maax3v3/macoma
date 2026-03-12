package web

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewAndRenderSuccess(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PreviewMaxDimension = 100
	h, err := Handler(cfg)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	src := createSamplePNG(t, 300, 200)

	previewReq := multipartRequest(t, "/api/preview", src, map[string]string{
		"delimiter_strategy": "border",
		"border_delimiter_color": "#000",
		"border_delimiter_tolerance": "10",
		"max_colors": "8",
	})
	previewRec := httptest.NewRecorder()
	h.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview status: got %d body=%s", previewRec.Code, previewRec.Body.String())
	}
	if ct := previewRec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("preview content-type: %q", ct)
	}
	previewImg := decodePNG(t, previewRec.Body.Bytes())
	if got := previewImg.Bounds().Dx(); got != 100 {
		t.Fatalf("preview width: got %d want 100", got)
	}

	renderReq := multipartRequest(t, "/api/render", src, map[string]string{
		"delimiter_strategy": "border",
		"border_delimiter_color": "#000",
		"border_delimiter_tolerance": "10",
		"max_colors": "8",
	})
	renderRec := httptest.NewRecorder()
	h.ServeHTTP(renderRec, renderReq)
	if renderRec.Code != http.StatusOK {
		t.Fatalf("render status: got %d body=%s", renderRec.Code, renderRec.Body.String())
	}
	renderImg := decodePNG(t, renderRec.Body.Bytes())
	if got := renderImg.Bounds().Dx(); got != 300 {
		t.Fatalf("render width: got %d want 300", got)
	}
}

func TestValidationErrors(t *testing.T) {
	cfg := DefaultConfig()
	h, err := Handler(cfg)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	tests := []struct {
		name       string
		req        *http.Request
		wantStatus int
	}{
		{
			name: "missing image",
			req: multipartNoFileRequest(t, "/api/preview", map[string]string{
				"delimiter_strategy": "color",
			}),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid strategy",
			req: multipartRequest(t, "/api/preview", createSamplePNG(t, 64, 64), map[string]string{
				"delimiter_strategy": "nope",
			}),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid tolerance",
			req: multipartRequest(t, "/api/preview", createSamplePNG(t, 64, 64), map[string]string{
				"color_delimiter_tolerance": "101",
			}),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "negative max colors",
			req: multipartRequest(t, "/api/preview", createSamplePNG(t, 64, 64), map[string]string{
				"max_colors": "-1",
			}),
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "unsupported image",
			req: multipartRequestWithContent(t, "/api/preview", "image", "bad.txt", []byte("not an image"), map[string]string{}),
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, tt.req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status: got %d want %d body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var payload map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("json parse: %v", err)
			}
			if payload["error"] == "" {
				t.Fatalf("expected error message")
			}
		})
	}
}

func TestBodyTooLarge(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxBodyBytes = 256
	h, err := Handler(cfg)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	large := bytes.Repeat([]byte{0}, 1024)
	req := multipartRequestWithContent(t, "/api/preview", "image", "big.bin", large, map[string]string{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status: got %d want %d body=%s", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
}

func TestStaticAndHealth(t *testing.T) {
	h, err := Handler(DefaultConfig())
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	h.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("health status: %d", healthRec.Code)
	}

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRec := httptest.NewRecorder()
	h.ServeHTTP(rootRec, rootReq)
	if rootRec.Code != http.StatusOK {
		t.Fatalf("root status: %d", rootRec.Code)
	}
	if !strings.Contains(rootRec.Body.String(), "Macoma") {
		t.Fatalf("root body missing title")
	}
	if !strings.Contains(rootRec.Body.String(), "integrity=\"sha384-") {
		t.Fatalf("root html missing SRI integrity attribute")
	}
	if !strings.Contains(rootRec.Body.String(), "crossorigin=\"anonymous\"") {
		t.Fatalf("root html missing crossorigin attribute")
	}
}

func multipartRequest(t *testing.T, target string, imageContent []byte, fields map[string]string) *http.Request {
	return multipartRequestWithContent(t, target, "image", "input.png", imageContent, fields)
}

func multipartNoFileRequest(t *testing.T, target string, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func multipartRequestWithContent(t *testing.T, target, fileField, fileName string, content []byte, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile(fileField, fileName)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write file: %v", err)
	}
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func createSamplePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	black := color.RGBA{0, 0, 0, 255}
	red := color.RGBA{255, 0, 0, 255}
	green := color.RGBA{0, 200, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}
	yellow := color.RGBA{255, 255, 0, 255}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			switch {
			case x < w/2 && y < h/2:
				img.SetRGBA(x, y, red)
			case x >= w/2 && y < h/2:
				img.SetRGBA(x, y, green)
			case x < w/2 && y >= h/2:
				img.SetRGBA(x, y, blue)
			default:
				img.SetRGBA(x, y, yellow)
			}
		}
	}
	for y := 0; y < h; y++ {
		img.SetRGBA(w/2, y, black)
	}
	for x := 0; x < w; x++ {
		img.SetRGBA(x, h/2, black)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func decodePNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	return img
}
