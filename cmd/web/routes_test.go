package main

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi"
)

var routes = []string{
	"/",
	"/login",
	"/logout",
	"/register",
	"/activate",
	"/members/plans",
	"/members/subscribe",
}

func Test_routes_exist(t *testing.T) {
	testToutes := testApp.routes()

	chiRoutes := testToutes.(chi.Router)

	for _, route := range routes {
		routeExist(t, chiRoutes, route)
	}
}

func routeExist(t *testing.T, chiRoutes chi.Router, route string) {
	routeFound := false

	_ = chi.Walk(chiRoutes, func(method string, foundRoute string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		if foundRoute == route {
			routeFound = true
		}
		return nil
	})

	if !routeFound {
		t.Errorf("route '%s' not found in registered routes", route)
	}
}
