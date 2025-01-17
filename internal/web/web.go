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
)

//go:embed templates/*
var templateFs embed.FS

func BuildRouter(interrogator *tagging.InterrogateForever) *chi.Mux {

	tmpl, err := template.ParseFS(templateFs, "templates/index.html")
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

		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Post("/api/v1/tag-image", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil { // Limit upload size to 10MB
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
				w.Header().Set("Content-Type", "application/json")
				if result.Error != nil {
					http.Error(w, result.Error.Error(), http.StatusInternalServerError)
					return
				}

				if err := json.NewEncoder(w).Encode(result.Tags); err != nil {
					http.Error(w, "failed to encode JSON", http.StatusInternalServerError)
				}

			}

		}

	})

	return r

}
