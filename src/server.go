package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Hello, World!")
	})

	if err := http.ListenAndServe(":8000", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
