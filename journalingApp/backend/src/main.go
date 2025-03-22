package main

import (
	"log"
	"net/http"
	"routes"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(wr http.ResponseWriter, req *http.Request) {
			wr.Header().Set("Access-Control-Allow-Origin", "*")
			wr.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			wr.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if req.Method == "OPTIONS" {
				wr.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(wr, req)
		},
	)
}

func main() {
	router := chi.NewRouter()

	router.Use(corsMiddleware)
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(render.SetContentType(render.ContentTypeJSON))

	router.Get("/", home)
	router.Route("/pages", routes.PageGlobalRoutes)

	log.Println("Starting app ...")
	http.ListenAndServe(":8080", router)
}

func home(writer http.ResponseWriter, response *http.Request) {
	writer.Write([]byte("Hello World"))
}
