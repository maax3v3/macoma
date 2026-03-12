package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/maax3v3/macoma/v2"
)

const (
	// PreviewMaxDimension controls the maximum width/height used by live preview.
	PreviewMaxDimension = 1200
	defaultMaxBodyBytes = 10 << 20 // 10MB
)

// Config configures the web server behavior.
type Config struct {
	MaxBodyBytes       int64
	RequestTimeout     time.Duration
	PreviewMaxDimension int
}

// DefaultConfig returns sensible defaults for web operation.
func DefaultConfig() Config {
	return Config{
		MaxBodyBytes:       defaultMaxBodyBytes,
		RequestTimeout:     30 * time.Second,
		PreviewMaxDimension: PreviewMaxDimension,
	}
}

// Handler builds an HTTP handler that serves the web UI and API.
func Handler(cfg Config) (http.Handler, error) {
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = defaultMaxBodyBytes
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 30 * time.Second
	}
	if cfg.PreviewMaxDimension <= 0 {
		cfg.PreviewMaxDimension = PreviewMaxDimension
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("loading static assets: %w", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.RequestTimeout))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/favicon.ico", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.Post("/api/preview", func(w http.ResponseWriter, r *http.Request) {
		serveConvert(w, r, cfg, true)
	})
	r.Post("/api/render", func(w http.ResponseWriter, r *http.Request) {
		serveConvert(w, r, cfg, false)
	})

	r.Handle("/*", http.FileServer(http.FS(staticSub)))

	return r, nil
}

func serveConvert(w http.ResponseWriter, r *http.Request, cfg Config, preview bool) {
	input, opts, err := parseRequest(w, r, cfg.MaxBodyBytes)
	if err != nil {
		writeError(w, err)
		return
	}

	if preview {
		input = scaleDown(input, cfg.PreviewMaxDimension)
	}

	out, err := macoma.Convert(input, opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("converting image: %v", err),
		})
		return
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("encoding png: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func parseRequest(w http.ResponseWriter, r *http.Request, maxBodyBytes int64) (image.Image, macoma.Options, error) {
	if r == nil {
		return nil, macoma.Options{}, badRequest("invalid request")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	if err := r.ParseMultipartForm(4 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) || strings.Contains(err.Error(), "request body too large") {
			return nil, macoma.Options{}, requestTooLarge("request body too large")
		}
		return nil, macoma.Options{}, badRequest(fmt.Sprintf("invalid multipart form: %v", err))
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		return nil, macoma.Options{}, badRequest("image is required")
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, macoma.Options{}, badRequest("unable to read image")
	}
	img, err := decodeImage(bytes.NewReader(data))
	if err != nil {
		return nil, macoma.Options{}, badRequest(fmt.Sprintf("invalid image: %v", err))
	}

	opts, err := optionsFromForm(r.MultipartForm.Value)
	if err != nil {
		return nil, macoma.Options{}, badRequest(err.Error())
	}

	return img, opts, nil
}

func optionsFromForm(values map[string][]string) (macoma.Options, error) {
	opts := macoma.DefaultOptions()

	get := func(key string) string {
		v := values[key]
		if len(v) == 0 {
			return ""
		}
		return v[0]
	}

	if strategy := get("delimiter_strategy"); strategy != "" {
		if strategy != macoma.StrategyColor && strategy != macoma.StrategyBorder {
			return opts, fmt.Errorf("delimiter_strategy must be %q or %q", macoma.StrategyColor, macoma.StrategyBorder)
		}
		opts.DelimiterStrategy = strategy
	}

	if hex := get("border_delimiter_color"); hex != "" {
		c, err := macoma.ParseHexColor(hex)
		if err != nil {
			return opts, fmt.Errorf("border_delimiter_color: %v", err)
		}
		opts.BorderDelimiterColor = c
	}

	if raw := get("border_delimiter_tolerance"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return opts, fmt.Errorf("border_delimiter_tolerance must be a number")
		}
		if v < 0 || v > 100 {
			return opts, fmt.Errorf("border_delimiter_tolerance must be between 0 and 100")
		}
		opts.BorderDelimiterTolerance = v
	}

	if raw := get("color_delimiter_tolerance"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return opts, fmt.Errorf("color_delimiter_tolerance must be a number")
		}
		if v < 0 || v > 100 {
			return opts, fmt.Errorf("color_delimiter_tolerance must be between 0 and 100")
		}
		opts.ColorDelimiterTolerance = v
	}

	if raw := get("max_colors"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return opts, fmt.Errorf("max_colors must be an integer")
		}
		if v < 0 {
			return opts, fmt.Errorf("max_colors must be >= 0")
		}
		opts.MaxColors = v
	}

	return opts, nil
}

type requestErr struct {
	status int
	msg    string
}

func (e requestErr) Error() string { return e.msg }

func badRequest(msg string) error {
	return requestErr{status: http.StatusBadRequest, msg: msg}
}

func requestTooLarge(msg string) error {
	return requestErr{status: http.StatusRequestEntityTooLarge, msg: msg}
}

func writeError(w http.ResponseWriter, err error) {
	var re requestErr
	if errors.As(err, &re) {
		writeJSON(w, re.status, map[string]string{"error": re.msg})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
