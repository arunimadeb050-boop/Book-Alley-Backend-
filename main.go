package main 

import(
	"fmt"
	"net/http"
)

func main (){
	connectDatabase()
	defer db.Close()


	// authentication and profile 
	http.HandleFunc("POST /register", registerHandler)
	http.HandleFunc("POST /login", loginHandler)
	http.HandleFunc("GET /profile", myProfileHandler)
	http.HandleFunc("PUT /profile", updateProfileHandler)
	http.HandleFunc("GET /users/{id}", userProfileHandler)

	// ---- Books: browse, search, filter ----
	http.HandleFunc("GET /books", getAllBooksHandler)
	http.HandleFunc("GET /books/{id}", getOneBookHandler)
	http.HandleFunc("POST /books", addBookHandler)
	http.HandleFunc("PUT /books/{id}", updateBookHandler)
	http.HandleFunc("DELETE /books/{id}", deleteBookHandler)
	http.HandleFunc("GET /my/books", myBooksHandler)
}