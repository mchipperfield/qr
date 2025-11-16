package qr

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"

	"golang.org/x/exp/slog"
)

type Response struct {
	Text  string
	Code  string
	Error string
}

type Handler struct {
	http.Handler
	logger *slog.Logger
	tmpl   *template.Template
}

func NewHandler(logger *slog.Logger) *Handler {
	tmpl := template.Must(template.New("").ParseFiles("qr.html"))
	h := Handler{
		logger: logger,
		tmpl:   tmpl,
	}

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", h.showForm(nil)).Methods(http.MethodGet)
	router.HandleFunc("/", h.generateCode()).Methods(http.MethodPost)
	h.Handler = router
	return &h
}

// GET /qr
func (h *Handler) showForm(data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h.tmpl.ExecuteTemplate(w, "qr", data); err != nil {
			h.logger.Error("failed to execute template", "error", err, "template", "qr")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// POST /qr
func (h *Handler) generateCode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			h.logger.Info("failed to parse form", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			h.showForm(Response{Error: "Something went wrong"}).ServeHTTP(w, r)
			return
		}

		encodedQR, err := GenerateGopherCode(r.Context(), CodeRequest{Text: r.PostFormValue("text"), DataType: r.PostFormValue("data_type")})
		if err != nil {
			h.logger.Info("failed to generate QR Code", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			h.showForm(Response{Error: "Something went wrong"}).ServeHTTP(w, r)
			return
		}

		h.showForm(Response{Text: r.PostFormValue("text"), Code: encodedQR}).ServeHTTP(w, r)
	}
}
