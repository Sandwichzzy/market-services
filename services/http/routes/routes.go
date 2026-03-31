package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/Sandwichzzy/market-services/services/http/service"
)

type Routes struct {
	router *chi.Mux
	srv    service.RestService
}

func NewRoutes(r *chi.Mux, srv service.RestService) Routes {
	return Routes{
		router: r,
		srv:    srv,
	}
}
