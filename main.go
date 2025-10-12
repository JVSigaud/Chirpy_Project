package main

import (
	"fmt"
	"net/http"
)

func main() {
	serveMux := http.NewServeMux()
	serveMux.Handle("/", http.FileServer(http.Dir(".")))
	var server = &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	fmt.Println("Server running at http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		// ListenAndServe always returns a non-nil error
		fmt.Println("Server error:", err)
	}
}
