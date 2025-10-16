package main

import "net/http"

func handlerReadiness(w http.ResponseWriter, rq *http.Request){
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func main(){
	mux := http.NewServeMux();
	server := http.Server{
		Handler: mux,
		Addr: ":8080",
	}
	fileServer := http.FileServer(http.Dir("."))
	mux.Handle("/app/", fileServer)
	mux.HandleFunc("/", handlerReadiness)
	
	server.ListenAndServe()

}