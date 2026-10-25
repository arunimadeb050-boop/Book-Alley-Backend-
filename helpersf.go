package main 

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"os"

	"github.com/golang-jwt/jwt/v5"
)


// Secret key used to sign JWT tokens.
// It is read from the JWT_SECRET environment variable, never typed in the code.
var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// sendJSON writes data back to the client as JSON.
func sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// sendError sends {"error": "message"}
func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, map[string]string{"error": message})
}

// readJSON reads the request body (JSON) into a struct you give it.
func readJSON(r *http.Request, target any) error {
	return json.NewDecoder(r.Body).Decode(target)
}

// getIDFromURL reads a number from the URL, e.g. /books/{id}
func getIDFromURL(r *http.Request, name string) (int, error) {
	idText := r.PathValue(name)
	id := 0
	valid := idText != ""

	for i := 0; i < len(idText); i++ {
		ch := idText[i]
		if ch < '0' || ch > '9' {
			valid = false
			break
		}
		id = id*10 + int(ch-'0')
	}

	if !valid {
		return 0, fmt.Errorf("id must be a number")
	}
	return id, nil
}

func now() string {
	return time.Now().Format("2006-01-02 15:04")
}

func today() string {
	return time.Now().Format("2006-01-02")
}

// createToken makes a JWT valid for 24 hours.
func createToken(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}


// getLoggedInUserID checks the "Authorization: Bearer <token>" header
// and returns the user's ID. Every protected endpoint starts by calling this.
func getLoggedInUserID(r *http.Request) (int, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return 0, fmt.Errorf("missing token, please login")
	}
	tokenString := strings.TrimPrefix(header, "Bearer ")

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid or expired token")
	}

	claims := token.Claims.(jwt.MapClaims)
	userID := int(claims["user_id"].(float64)) // JSON numbers are float64
	return userID, nil
}

// withCORS lets a frontend on another port (like :3000 or :5500) talk to us.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if r.Method == http.MethodOptions { // browser "preflight" check
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}


