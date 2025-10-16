package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
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

func (cfg *apiConfig) handlerReqNumber(w http.ResponseWriter, rq *http.Request){
	w.Write([]byte(fmt.Sprintf("Hits: %d\n",cfg.fileserverHits.Load())))
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
	mux.HandleFunc("/", handlerReadiness)
	mux.HandleFunc("/metrics",apiCfg.handlerReqNumber)
	mux.HandleFunc("/reset", apiCfg.handlerResetReqNum)
	server.ListenAndServe()

}