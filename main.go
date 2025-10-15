package main

import (
	"encoding/json"
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf(
		`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))
}
func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)

}

func handler(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Body string `json:"body"`
	}
	type responseParams struct {
		Err   string `json:"error"`
		Valid bool   `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	var p params
	var respBody responseParams

	err := decoder.Decode(&p)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		respBody.Err = "Something went wrong"
		respBody.Valid = false
		w.WriteHeader(500)
	}
	if len(p.Body) > 140 {
		respBody.Err = "Chirp is too long"
		respBody.Valid = false
		w.WriteHeader(400)

	} else {
		respBody.Err = "nil"
		respBody.Valid = true
		w.WriteHeader(200)
	}
	data, err := json.Marshal(respBody)

	if err != nil {
		respBody.Err = "Something went wrong"
		respBody.Valid = false
		w.WriteHeader(500)
	}

	w.Write(data)

}

func main() {
	apicfg := &apiConfig{}
	serveMux := http.NewServeMux()
	serveMux.Handle("/app/", apicfg.middlewareMetricsInc(
		http.StripPrefix("/app", http.FileServer(http.Dir(".")))))

	serveMux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	serveMux.HandleFunc("GET /admin/metrics", apicfg.handleMetrics)
	serveMux.HandleFunc("POST /admin/reset", apicfg.resetMetrics)

	serveMux.HandleFunc("POST /api/validate_chirp", handler)

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
