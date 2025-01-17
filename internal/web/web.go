package web

import (
	"embed"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"html/template"
	"imagetag/internal/tagging"
	"log"
	"net/http"
	"strings"
)

//go:embed templates/*
var templateFs embed.FS

func BuildRouter(interrogator *tagging.InterrogateForever) *chi.Mux {

	indexTmpl, err := template.ParseFS(templateFs, "templates/index.html")
	if err != nil {
		log.Panicf("Error parsing templates: %v", err)
	}
	respTmpl, err := template.ParseFS(templateFs, "templates/response.html")
	if err != nil {
		log.Panicf("Error parsing templates: %v", err)
	}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(middleware.Compress(6))
	r.Use(middleware.StripSlashes)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
		}{}

		if err := indexTmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	handleResults := func(w http.ResponseWriter, r *http.Request, result tagging.JobResult) {
		acceptHeader := r.Header.Get("Accept")

		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("X-XSS-Protection", "1")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if acceptsJson(acceptHeader) {
			w.Header().Set("Content-Type", "application/json")
			if result.Error != nil {
				http.Error(w, result.Error.Error(), http.StatusInternalServerError)
				return
			}

			if err := json.NewEncoder(w).Encode(result.Tags); err != nil {
				http.Error(w, "failed to encode JSON", http.StatusInternalServerError)
				return
			}
		} else {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			data := struct {
				Tags []string
			}{
				Tags: result.Tags,
			}

			if err := respTmpl.Execute(w, data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

	}

	r.Post("/api/v1/tag-image", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil { // Limit memory usage to 10MB
			http.Error(w, "File too big or malformed", http.StatusBadRequest)
			return
		}

		file, fileHeader, err := r.FormFile("image")
		log.Printf("received file: %v", fileHeader.Filename)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer file.Close()

		c, cancel, err := interrogator.TagImage(file)
		if err != nil {
			log.Printf("Error creating tag image: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		for {
			select {
			case <-r.Context().Done():
				cancel()
				log.Println("client disconnected")
				http.Error(w, "Client disconnected", http.StatusRequestTimeout)
				return
			case result := <-c:
				log.Printf("job result: %v", result)
				if result.Error != nil {
					http.Error(w, result.Error.Error(), http.StatusInternalServerError)
					return
				}
				handleResults(w, r, result)
				return

			}

		}

	})

	return r

}

func acceptsJson(acceptHeader string) bool {
	parts := strings.Split(acceptHeader, ",")
	for _, part := range parts {
		if part == "*/*" {
			return true
		}
		if strings.Contains(part, "application/json") {
			return true
		}
	}
	return false
}
