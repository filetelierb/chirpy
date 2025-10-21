package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	//"github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler{
	newHandlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
	return newHandlerFunc;
}

func handlerReadiness(w http.ResponseWriter, rq *http.Request){
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func handlerVerifyChirp(w http.ResponseWriter, rq *http.Request){
	type RequestBody struct{
		Body string `json:"body"`
	}

	type returnVals struct {
		Error string		`json:"error"`
		CleanedBody string	`json:"cleaned_body"`
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
	w.Write(data);
}

func (cfg *apiConfig) handlerReqNumber(w http.ResponseWriter, rq *http.Request){
	w.Header().Add("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`,cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) handlerResetReqNum(w http.ResponseWriter, rq *http.Request){
	w.Write([]byte("Counter set to 0"))
	cfg.fileserverHits.Store(0)
}

func main(){
	mux := http.NewServeMux();
	server := http.Server{
		Handler: mux,
		Addr: ":8080",
	}
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	apiCfg.fileserverHits.Store(0)
	fileServer := http.FileServer(http.Dir("."))
	fileServer = http.StripPrefix("/app", fileServer)
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileServer))
	mux.HandleFunc("GET /admin/metrics",apiCfg.handlerReqNumber)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerResetReqNum)
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /api/validate_chirp", handlerVerifyChirp)
	server.ListenAndServe()

}