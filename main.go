package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"Chirpy_Project/internal/database"

	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirpy struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func mapChirpy(chirpSlice []database.Chirp) []Chirpy {
	chirpySlice := make([]Chirpy, len(chirpSlice))
	for idx, chirp := range chirpSlice {
		chirpySlice[idx] = Chirpy{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}

	}
	return chirpySlice
}

// func clean_body(body string) string {
// 	bads := map[string]struct{}{
// 		"kerfuffle": {},
// 		"sharbert":  {},
// 		"fornax":    {},
// 	}

// 	words := strings.Split(body, " ")
// 	cleaned := make([]string, 0, len(words))

// 	for _, w := range words {
// 		if _, ok := bads[strings.ToLower(w)]; ok {
// 			cleaned = append(cleaned, "****")
// 		} else {
// 			cleaned = append(cleaned, w)
// 		}
// 	}
// 	return strings.Join(cleaned, " ")
// }

type apiConfig struct {
	fileserverHits atomic.Int32
	objQuery       *database.Queries
	Platform       string
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
	if cfg.Platform != "dev" {
		w.WriteHeader(403)
		return
	}
	cfg.objQuery.Reset(r.Context())
	w.WriteHeader(200)
}

func (cfg *apiConfig) CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	type params struct {
		Email string `json:"email"`
	}
	var p params
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&p)
	if err != nil {
		w.WriteHeader(500)

		return
	}

	user, err := cfg.objQuery.CreateUser(r.Context(), p.Email)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var usr User = User{ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email}

	data, _ := json.Marshal(usr)

	w.WriteHeader(201)
	w.Write(data)

}

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {

	type params struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	// type responseParams struct {
	// 	Err         string `json:"error"`
	// 	Valid       bool   `json:"valid"`
	// 	CleanedBody string `json:"cleaned_body"`
	// }

	decoder := json.NewDecoder(r.Body)
	var p params
	// var respBody responseParams

	err := decoder.Decode(&p)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		// respBody.Err = "Something went wrong"
		// respBody.Valid = false
		w.WriteHeader(500)
		return
	}
	if len(p.Body) > 140 {
		// respBody.Err = "Chirp is too long"
		// respBody.Valid = false

		w.WriteHeader(400)
		return

	} else {
		// respBody.Err = ""
		// respBody.Valid = true
		// respBody.CleanedBody = clean_body(p.Body)
		w.WriteHeader(201)
	}
	// parsedUserID, err := uuid.Parse(reqBody.UserID)

	chirpParams := database.CreateChirpParams{
		Body:   p.Body,
		UserID: p.UserID,
	}

	user, err := cfg.objQuery.CreateChirp(r.Context(), chirpParams)
	// fmt.Printf("User object from CreateChirp: %+v\n", user)

	if err != nil {
		w.WriteHeader(500)
		return
	}
	nwUser := Chirpy{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Body:      user.Body,
		UserID:    user.UserID,
	}

	data, err := json.Marshal(nwUser)

	if err != nil {
		// respBody.Err = "Something went wrong"
		// respBody.Valid = false
		w.WriteHeader(500)
		return
	}

	w.Write(data)

}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	user, err := cfg.objQuery.GetChirps(r.Context())
	if err != nil {
		w.WriteHeader(500)
		return
	}
	dataChirp := mapChirpy(user)
	data, err := json.Marshal(dataChirp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write(data)

}
func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	chirp, err := cfg.objQuery.GetChirp(r.Context(), id)
	if err != nil {
		w.WriteHeader(404)
		return
	}
	dat, err := json.Marshal(Chirpy{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	})
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write(dat)
}
func main() {
	godotenv.Load(".env")
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)

	apicfg := &apiConfig{objQuery: dbQueries, Platform: platform}

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
	serveMux.HandleFunc("POST /api/users", apicfg.CreateUser)

	serveMux.HandleFunc("POST /api/chirps", apicfg.handlerChirps)
	serveMux.HandleFunc("GET /api/chirps", apicfg.handlerGetChirps)
	serveMux.HandleFunc("GET /api/chirps/{ID}", apicfg.handlerGetChirp)
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
