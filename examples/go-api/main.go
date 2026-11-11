// A tiny HTTP API used to demo dockopt. The Dockerfile next to it has the
// usual problems; run `dockopt bench examples/go-api` to see what fixing them buys.
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var greetings = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "greetings_total",
	Help: "Greetings served, by name.",
}, []string{"name"})

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok\n"))
	})
	r.Get("/hello/{name}", func(w http.ResponseWriter, req *http.Request) {
		name := chi.URLParam(req, "name")
		greetings.WithLabelValues(name).Inc()
		w.Write([]byte("hello " + name + "\n"))
	})
	r.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", r))
}
