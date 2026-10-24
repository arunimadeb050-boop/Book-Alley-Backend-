package main 

import(
	"fmt"
	"net/http"
)

func main (){
	


	// authentication and profile 
	http.HandleFunc("POST /register", registerHandler)
	http.HandleFunc("POST /login", loginHandler)
	http.HandleFunc("GET /profile", myProfileHandler)
	http.HandleFunc("PUT /profile", updateProfileHandler)
	http.HandleFunc("GET /users/{id}", userProfileHandler)

	
}