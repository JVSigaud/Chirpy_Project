package main

import (
	"fmt"
	"net/http"
)

func main() {
	serveMux := http.NewServeMux()
	var server = &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	err := server.ListenAndServe()

	if err != nil {
		fmt.Sprintf("Erro: %v", err)
	}

}
