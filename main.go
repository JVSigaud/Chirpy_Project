package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	// ...
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("Hits: " + fmt.Sprintf("%d", cfg.fileserverHits.Load())))
}
func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)

}

func main() {
	apicfg := &apiConfig{}
	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", apicfg.middlewareMetricsInc(
		http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	serveMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	serveMux.HandleFunc("/metrics", apicfg.handleMetrics)
	serveMux.HandleFunc("/reset", apicfg.resetMetrics)

	var server = &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	fmt.Println("Server running at http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		// ListenAndServe always returns a non-nil erro
		// r
		fmt.Println("Server error:", err)
	}
}
