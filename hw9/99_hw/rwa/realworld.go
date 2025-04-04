package main

import (
	"fmt"
	"io"
	"net/http"
)

// сюда писать код

// POST /users/ (register)
// POST /users/login
// GET /user
// UPDATE /user
// POST /articles Create Article
// GET /articles Get Articles
// GET /articles?author= Articles by Author
// GET /articles?tag= Articles by tag
func GetApp() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", Get)
	mux.HandleFunc("/api/users", Register)
	mux.HandleFunc("/api/users/login", Login)

	return mux
}

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Println(r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	m := make(map[string]interface{})
	body, bytesErr := io.ReadAll(r.Body)
	if bytesErr != nil {
		panic(bytesErr)
	}
	fmt.Println(string(body))

	fmt.Println(m)

	w.WriteHeader(201)
}

func Get(w http.ResponseWriter, r *http.Request) {

}

func Login(w http.ResponseWriter, r *http.Request) {

}
