package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mchipperfield/qr"
	"golang.org/x/exp/slog"
)

type Handler struct {
	http.Handler
	logger *slog.Logger
}

func NewHandler(l *slog.Logger) *Handler {
	h := Handler{
		logger: l,
	}
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/qrcode", h.GenerateCode())
	router.Use(CheckContentHeader("application/json"))
	h.Handler = router
	return &h
}

// POST /qrcode
func (h *Handler) GenerateCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req qr.CodeRequest
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		encodedQR, err := qr.GenerateCode(r.Context(), req)
		if err != nil {
			switch {
			case errors.Is(err, qr.ErrUnsupportedDataType), errors.Is(err, qr.ErrRequired):
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			default:
				h.logger.Info("failed to generate QR Code", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(encodedQR)
	}
}

func CheckContentHeader(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			if header := r.Header.Get("Content-Type"); header != "" && header != contentType {
				w.WriteHeader(http.StatusUnsupportedMediaType)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
