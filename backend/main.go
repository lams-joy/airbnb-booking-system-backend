package main

import (
"fmt"
"log"
"net/http"
"os"

"github.com/go-chi/chi/v5"
"github.com/go-chi/chi/v5/middleware"

"airbnb-booking-system/db"
"airbnb-booking-system/handlers"
)

func main() {
db.InitDB()
db.InitRedis()

r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)

r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(`{"status": "ok", "app": "airbnb-booking-system"}`))
})

r.Route("/api/properties", func(r chi.Router) {
r.Post("/", handlers.CreateProperty)
r.Get("/search", handlers.SearchProperties)
r.Get("/{id}", handlers.GetPropertyByID)
})

port := os.Getenv("PORT")
if port == "" {
port = "8080"
}

fmt.Printf("?? Server running on port %s...\n", port)
log.Fatal(http.ListenAndServe(":"+port, r))
}
