package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"database/sql"

	"github.com/filetelierb/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	newHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
	return newHandlerFunc
}

func handlerReadiness(w http.ResponseWriter, rq *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func handlerVerifyChirp(w http.ResponseWriter, rq *http.Request) {
	type RequestBody struct {
		Body string `json:"body"`
	}

	type returnVals struct {
		Error       string `json:"error"`
		CleanedBody string `json:"cleaned_body"`
	}

	respBody := returnVals{}
	decoder := json.NewDecoder(rq.Body)
	reqB := RequestBody{}
	err := decoder.Decode(&reqB)
	if err != nil {
		log.Printf("Something went wrong")
		w.WriteHeader(500)
		return
	}

	if len(reqB.Body) <= 140 {
		badWords := [3]string{
			"kerfuffle",
			"sharbert",
			"fornax",
		}
		respBody.CleanedBody = reqB.Body

		for _, bw := range badWords {
			repWord := "****"
			bw2 := strings.ToUpper(bw[:1]) + bw[1:]
			respBody.CleanedBody = strings.ReplaceAll(respBody.CleanedBody, bw, repWord)
			respBody.CleanedBody = strings.ReplaceAll(respBody.CleanedBody, bw2, repWord)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
	} else {
		respBody.Error = "Chirp is too long"
		w.WriteHeader(400)
	}
	data, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Something went wrong")
		w.WriteHeader(500)
		return
	}
	w.Write(data)
}

func (cfg *apiConfig) handlerReqNumber(w http.ResponseWriter, rq *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerResetReqNum(w http.ResponseWriter, rq *http.Request) {
	w.Write([]byte("Counter set to 0"))
	err := cfg.db.ClearUserTable(rq.Context())
	if err != nil {
		log.Printf("Something went wrong clearing the user table")
		w.WriteHeader(500)
		return
	}
	if os.Getenv("PLATFORM") != "dev" {
		w.WriteHeader(403)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	cfg.fileserverHits.Store(0)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, rq *http.Request) {
	type RequestBody struct {
		Email string `json:"email"`
	}

	type responseValue struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt int       `json:"created_at"`
		UpdatedAt int       `json:"updated_at"`
		Email     string    `json:"email"`
	}
	decoder := json.NewDecoder(rq.Body)
	rBody := RequestBody{}
	err := decoder.Decode(&rBody)
	if err != nil {
		log.Printf("Something went wrong decoding the body")
		w.WriteHeader(500)
		return
	}
	newUser, err := cfg.db.CreateUser(
		rq.Context(),
		database.CreateUserParams{
			ID:    uuid.New(),
			Email: rBody.Email,
		})
	if err != nil {
		log.Printf("Something went wrong creating the user: %v", err)
		w.WriteHeader(500)
		return
	}

	// Convert database result to response format
	response := responseValue{
		ID:        newUser.ID,
		CreatedAt: int(newUser.CreatedAt.Unix()),
		UpdatedAt: int(newUser.UpdatedAt.Unix()),
		Email:     newUser.Email,
	}

	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Something went wrong creating the user")
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)

}

func (cfg *apiConfig) handlerCreateChirpy(w http.ResponseWriter, rq *http.Request){
	type RequestBody struct{
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type cleanedBody struct {
		Error       string `json:"error"`
		CleanedBody string `json:"cleaned_body"`
	}

	type responseValue struct{
		ID uuid.UUID `json:"id"`
		CreatedAt int `json:"created_at"`
		UpdatedAt int `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	rBody := RequestBody{}
	decoder := json.NewDecoder(rq.Body)
	err := decoder.Decode(&rBody)
	if err != nil {
		fmt.Printf("Error decoding the body: %v", err)
		w.WriteHeader(500)
		return
	}

	respBody := cleanedBody{}

	if len(rBody.Body) <= 140 {
		badWords := [3]string{
			"kerfuffle",
			"sharbert",
			"fornax",
		}
		respBody.CleanedBody = rBody.Body

		for _, bw := range badWords {
			repWord := "****"
			bw2 := strings.ToUpper(bw[:1]) + bw[1:]
			respBody.CleanedBody = strings.ReplaceAll(respBody.CleanedBody, bw, repWord)
			respBody.CleanedBody = strings.ReplaceAll(respBody.CleanedBody, bw2, repWord)
		}
	} else {
		respBody.Error = "Chirp is too long"
		w.WriteHeader(400)
		return
	}

	db := cfg.db
	chirp, err := db.CreateChirp(rq.Context(), database.CreateChirpParams{
		ID: uuid.New(),
		Body: respBody.CleanedBody,
		UserID: uuid.NullUUID{
			UUID: rBody.UserID,
			Valid: true,
		},
	})
	if err != nil {
		fmt.Printf("Error creating the chirp: %v", err)
		w.WriteHeader(500)
		return
	}
	rValue := responseValue{
		ID: chirp.ID,
		CreatedAt: int(chirp.CreatedAt.UnixMilli()),
		UpdatedAt: int(chirp.UpdatedAt.UnixMilli()),
		Body: chirp.Body,
		UserID: chirp.UserID.UUID,
	}
	data, err := json.Marshal(rValue)
	if err != nil {
		fmt.Printf("Error encoding the new chirp: %v", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(201)
	w.Header().Add("Content-Type","application/json")
	w.Write(data)

}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, rq *http.Request){
	type responsePayload struct {
		ID uuid.UUID `json:"id"`
		CreatedAt int `json:"created_at"`
		UpdatedAt int `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	db := cfg.db

	chirps, err := db.GetChirps(rq.Context())
	if err != nil{
		fmt.Printf("Error fetching chirps: %v", err)
		w.WriteHeader(500)
		return
	}
	var resp []responsePayload
	for _, chirp := range chirps {
		resp = append(resp,  responsePayload{
			ID: chirp.ID,
			CreatedAt: int(chirp.CreatedAt.UnixMilli()),
			UpdatedAt: int(chirp.UpdatedAt.UnixMilli()),
			Body: chirp.Body,
			UserID: chirp.UserID.UUID,
			
		})
	}
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Printf("Error encoding chirps: %v", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Header().Add("Content-Type","application/json")
	w.Write(data)
}

func (cfg *apiConfig) handlerGetSingleChirp(w http.ResponseWriter, rq *http.Request){
	type responsePayload struct {
		ID uuid.UUID `json:"id"`
		CreatedAt int `json:"created_at"`
		UpdatedAt int `json:"updated_at"`
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	db := cfg.db

	chirp, err := db.GetSingleChirp(rq.Context(), uuid.MustParse(rq.PathValue("chirpID")) )
	if err != nil {
		w.WriteHeader(404)
		return
	}

	resp := responsePayload{
		ID: chirp.ID,
		CreatedAt: int(chirp.CreatedAt.UnixMilli()),
		UpdatedAt: int(chirp.CreatedAt.UnixMilli()),
		Body: chirp.Body,
		UserID: chirp.UserID.UUID,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Header().Add("Content-Type","application/json")
	w.Write(data)
}

func main() {
	godotenv.Load()
	mux := http.NewServeMux()
	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		fmt.Printf("Error connecting to the database: %v", err)
		os.Exit(1)
	}
	dbQueries := database.New(db)
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQueries,
	}
	apiCfg.fileserverHits.Store(0)
	fileServer := http.FileServer(http.Dir("."))
	fileServer = http.StripPrefix("/app", fileServer)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileServer))
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerReqNumber)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetReqNum)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/validate_chirp", handlerVerifyChirp)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirpy)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}",apiCfg.handlerGetSingleChirp)
	server.ListenAndServe()

}
