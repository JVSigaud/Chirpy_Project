package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"

	"Chirpy_Project/internal/auth"
	"Chirpy_Project/internal/database"

	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"hashed_password"`
	Token          string    `json:"token"`
	RefreshToken   string    `json:"refresh_token"`
	IsChirpyRed    bool      `json:"is_chirpy_red"`
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

func clean_body(body string) string {
	bads := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}

	words := strings.Split(body, " ")
	cleaned := make([]string, 0, len(words))

	for _, w := range words {
		if _, ok := bads[strings.ToLower(w)]; ok {
			cleaned = append(cleaned, "****")
		} else {
			cleaned = append(cleaned, w)
		}
	}
	return strings.Join(cleaned, " ")
}

type apiConfig struct {
	fileserverHits atomic.Int32
	objQuery       *database.Queries
	Platform       string
	keyJWT         string
	polkaKey       string
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
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var p params
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&p)
	if err != nil {
		w.WriteHeader(500)

		return
	}
	hashed, _ := auth.HashPassword(p.Password)
	user, err := cfg.objQuery.CreateUser(
		r.Context(), database.CreateUserParams{
			Email: p.Email, HashedPassword: hashed})
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var usr User = User{ID: user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}

	data, _ := json.Marshal(usr)

	w.WriteHeader(201)
	w.Write(data)

}

func (cfg *apiConfig) handlerUserLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	type params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var p params
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		w.WriteHeader(500)
		return
	}

	user, err := cfg.objQuery.GetUser(r.Context(), p.Email)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	is_valid, err := auth.CheckPasswordHash(p.Password, user.HashedPassword)

	if is_valid && err == nil {
		var token string

		token, _ = auth.MakeJWT(user.ID, cfg.keyJWT, time.Duration(60)*time.Minute)
		refreshToken, err := auth.MakeRefreshToken()
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		err = cfg.objQuery.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{Token: refreshToken, UserID: user.ID})
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		// fmt.Println(token)
		dat, err := json.Marshal(User{
			ID:           user.ID,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
			Email:        user.Email,
			IsChirpyRed:  user.IsChirpyRed,
			Token:        token,
			RefreshToken: refreshToken,
		})
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Write(dat)
	} else {
		w.WriteHeader(401)
		return
	}

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
	tokenAuth, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte(err.Error()))
		return
	}
	id, err := auth.ValidateJWT(tokenAuth, cfg.keyJWT)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("401 unauthorized"))
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
		p.Body = clean_body(p.Body)
		w.WriteHeader(201)
	}
	// parsedUserID, err := uuid.Parse(reqBody.UserID)

	chirpParams := database.CreateChirpParams{
		Body:   p.Body,
		UserID: id,
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
func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	user, err := cfg.objQuery.LookUpToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.keyJWT,
		time.Hour,
	)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}
func (cfg *apiConfig) handlerPostUsers(w http.ResponseWriter, r *http.Request) {
	type req struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	var p req
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&p)

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Not able to get token", err)
		return
	}
	// user, err := cfg.objQuery.LookUpToken(r.Context(), token)
	userID, err := auth.ValidateJWT(token, cfg.keyJWT)
	if err != nil {
		respondWithError(w, 401, "token malformed", err)
		return
	}

	hashPassword, err := auth.HashPassword(p.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create a new password", err)
		return
	}
	// user.HashedPassword = hashPassword
	// user.Email = p.Email
	update, err := cfg.objQuery.UpdateUser(
		r.Context(),
		database.UpdateUserParams{ID: userID,
			Email:          p.Email,
			HashedPassword: hashPassword})
	if err != nil {
		respondWithError(w, 401, "not able to update", err)
		return
	}
	dat := User{
		Email:       update.Email,
		ID:          update.ID,
		CreatedAt:   update.CreatedAt,
		UpdatedAt:   update.UpdatedAt,
		IsChirpyRed: update.IsChirpyRed,
	}
	respondWithJSON(w, http.StatusOK, dat)

}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	_, err = cfg.objQuery.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter,r *http.Request){

// }
func (cfg *apiConfig) GetChirpsByUsers(r *http.Request) ([]database.Chirp, error) {
	s := r.URL.Query().Get("author_id")
	if s == "" {
		user, err := cfg.objQuery.GetChirps(r.Context())
		return user, err

	} else {
		uiid, _ := uuid.Parse(s)
		user, err := cfg.objQuery.GetChirpsByUserId(r.Context(), uiid)
		return user, err
	}

}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	user, err := cfg.GetChirpsByUsers(r)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	order := r.URL.Query().Get("sort")
	fmt.Println(order)
	if order == "desc" {
		sort.Slice(user, func(i, j int) bool { return user[i].CreatedAt.After(user[j].CreatedAt) })
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

func (cfg *apiConfig) handleDeleteChirp(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Not able to get token", err)
		return
	}
	// user, err := cfg.objQuery.LookUpToken(r.Context(), token)
	userID, err := auth.ValidateJWT(token, cfg.keyJWT)
	if err != nil {
		respondWithError(w, 401, "token malformed", err)
		return
	}
	id, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		respondWithError(w, 401, "Can`t get Id", err)
		return
	}
	user, err := cfg.objQuery.GetChirp(r.Context(), id)
	if err != nil {
		respondWithError(w, 404, "chirp not found", err)
		return
	}
	if user.UserID != userID {
		respondWithError(w, 403, "Not alowed", errors.New("unauthorized access"))
		return
	}

	if cfg.objQuery.DeleteChirp(r.Context(), database.DeleteChirpParams{UserID: userID, ID: id}) != nil {
		respondWithError(w, 403, "Not authorized", err)
		return
	}

	respondWithJSON(w, 204, true)
}

func (cfg *apiConfig) handleWebHook(w http.ResponseWriter, r *http.Request) {
	type dat struct {
		UserId string `json:"user_id"`
	}
	type req struct {
		Event string `json:"event"`
		Data  dat    `json:"data"`
	}
	type empty struct{}
	var emp empty
	var p req
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&p)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "couldnt decode", err)
		return
	}

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, 401, "cant get api key", err)
		return
	}

	if !(apiKey == cfg.polkaKey) {
		respondWithError(w, 401, "apikey not correct", errors.New("not the polka api key"))
		return
	}

	if p.Event == "user.upgraded" {
		uuid, _ := uuid.Parse(p.Data.UserId)
		err := cfg.objQuery.UpgradeMember(r.Context(), uuid)
		if err != nil {
			respondWithError(w, 404, "Coudnt find user id", err)
			return
		}
		respondWithJSON(w, 204, emp)

	} else {
		respondWithJSON(w, 204, emp)
	}

}
func main() {
	godotenv.Load(".env")
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	key := os.Getenv("JWTs_KEY")
	polkaKey := os.Getenv("POLKA_KEY")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)

	apicfg := &apiConfig{objQuery: dbQueries, Platform: platform, keyJWT: key, polkaKey: polkaKey}

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
	serveMux.HandleFunc("POST /api/login", apicfg.handlerUserLogin)

	serveMux.HandleFunc("POST /api/chirps", apicfg.handlerChirps)
	serveMux.HandleFunc("GET /api/chirps", apicfg.handlerGetChirps)
	serveMux.HandleFunc("GET /api/chirps/{ID}", apicfg.handlerGetChirp)
	serveMux.HandleFunc("DELETE /api/chirps/{ID}", apicfg.handleDeleteChirp)

	serveMux.HandleFunc("POST /api/refresh", apicfg.handlerRefresh)
	serveMux.HandleFunc("POST /api/revoke", apicfg.handlerRevoke)
	serveMux.HandleFunc("PUT /api/users", apicfg.handlerPostUsers)

	serveMux.HandleFunc("POST /api/polka/webhooks", apicfg.handleWebHook)

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
