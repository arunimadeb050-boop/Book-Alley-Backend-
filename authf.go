package main

// handles registration of the user , authenctication and profile management 

import (
	"math"
	"net/http"
	"strings"
	"encoding/json"

	"golang.org/x/crypto/bcrypt"
)

//POST /register
// Body : {"name": "Arunima", "email": "Arunima@gmail.com", "password": "secret123"}
func registerHandler(w http.ResponseWriter, r *http.Request){
	type userinput struct{
		Name string `json:"name"`
		Email string `json:"email"`
		Password string `json:"password"`
	}

	var input userinput

	if err:=json.NewDecoder(r.Body).Decode(&input);err!=nil{
		sendError(w,http.StatusBadRequest,"invalid JSON")
		return
	}

	//validate the input 
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	if input.Name == "" || input.Email == "" {
		sendError(w, http.StatusBadRequest, "name and email are required")
		return
	}
	email := input.Email
	found := false 

	for i := 0; i < len(email); i++ {
    	if email[i] == '@' {
        	found = true 
        	break        
    	}
	}

	if found == false {
    	sendError(w, http.StatusBadRequest, "email looks invalid")
    	return
	}
	if len(input.Password) < 6 {
		sendError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}


	var count int 
	//if the email is already in use or not 
	db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", input.Email).Scan(&count)
	if count > 0 {
		sendError(w, http.StatusConflict, "email already registered")
		return
	}
	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "could not hash password")
		return
	}


	result, err := db.Exec("INSERT INTO users (name, email, password) VALUES (?, ?, ?)",
		input.Name, input.Email, string(hash))
	if err != nil {
		sendError(w, http.StatusInternalServerError, "could not create user")
		return
	}
	id, _ := result.LastInsertId()
	sendJSON(w, http.StatusCreated, User{ID: int(id), Name: input.Name, Email: input.Email})

}



// POST /login
// Body: {"email": "Arunima@mail.com", "password": "secret123"}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &input); err != nil {
		sendError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))

	// Find the user by email
	var u User
	err := db.QueryRow("SELECT id, name, email, password, bio, city FROM users WHERE email = ?",
		input.Email).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Bio, &u.City)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "wrong email or password")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(input.Password)) != nil {
		sendError(w, http.StatusUnauthorized, "wrong email or password")
		return
	}

	token, err := createToken(u.ID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "could not create token")
		return
	}
	sendJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

// buildProfile collects a user's info + their rating + their reviews.
// showEmail is true only when you look at your OWN profile.
func buildProfile(userID int, showEmail bool) (map[string]any, error) {
	var u User
	err := db.QueryRow("SELECT id, name, email, bio, city FROM users WHERE id = ?",
		userID).Scan(&u.ID, &u.Name, &u.Email, &u.Bio, &u.City)
	if err != nil {
		return nil, err
	}
	if !showEmail {
		u.Email = "" 
	}

	// Reputation = average of all star ratings
	var average float64
	var count int
	db.QueryRow("SELECT COALESCE(AVG(rating), 0), COUNT(*) FROM reviews WHERE reviewed_id = ?",
		userID).Scan(&average, &count)

	// Reviews others wrote about this user
	rows, err := db.Query(`SELECT r.rating, r.comment, u.name, r.created_at
		FROM reviews r JOIN users u ON u.id = r.reviewer_id
		WHERE r.reviewed_id = ? ORDER BY r.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []Review{}
	for rows.Next() {
		var rv Review
		rows.Scan(&rv.Rating, &rv.Comment, &rv.ReviewerName, &rv.CreatedAt)
		reviews = append(reviews, rv)
	}

	return map[string]any{
		"user":           u,
		"average_rating": math.Round(average*10) / 10, // 1 decimal place
		"review_count":   count,
		"reviews":        reviews,
	}, nil
}

// GET /profile  -> my own profile (login needed)
func myProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getLoggedInUserID(r)
	if err != nil {
		sendError(w, http.StatusUnauthorized, err.Error())
		return
	}
	profile, err := buildProfile(userID, true)
	if err != nil {
		sendError(w, http.StatusNotFound, "user not found")
		return
	}
	sendJSON(w, http.StatusOK, profile)
}

// GET /users/{id}  -> anyone's public profile (with rating & reviews)
func userProfileHandler(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromURL(r, "id")
	if err != nil {
		sendError(w, http.StatusBadRequest, "user id must be a number")
		return
	}
	profile, err := buildProfile(id, false)
	if err != nil {
		sendError(w, http.StatusNotFound, "user not found")
		return
	}
	sendJSON(w, http.StatusOK, profile)
}

// PUT /profile
// Body: {"name": "Arunima", "bio": "I love sci-fi", "city": "Moulvibazar"}
func updateProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getLoggedInUserID(r)
	if err != nil {
		sendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var input struct {
		Name string `json:"name"`
		Bio  string `json:"bio"`
		City string `json:"city"`
	}
	if err := readJSON(r, &input); err != nil {
		sendError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		sendError(w, http.StatusBadRequest, "name cannot be empty")
		return
	}

	_, err = db.Exec("UPDATE users SET name = ?, bio = ?, city = ? WHERE id = ?",
		input.Name, input.Bio, input.City, userID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "could not update profile")
		return
	}

	profile, _ := buildProfile(userID, true)
	sendJSON(w, http.StatusOK, profile)
}
