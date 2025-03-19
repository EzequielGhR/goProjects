module main

go 1.24.0

require (
	github.com/go-chi/chi/v5 v5.2.1
	github.com/go-chi/render v1.0.3
	routes v0.0.0-00010101000000-000000000000
)

require (
	github.com/ajg/form v1.5.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	models v0.0.0-00010101000000-000000000000 // indirect
)

replace models => ./models

replace routes => ./routes
