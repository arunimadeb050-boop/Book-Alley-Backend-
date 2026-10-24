package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
)

//columns needed for books and b is a short form for books table 
const bookColumns="b.id, b.title, b.author , b.genre, b.description, b.owner_id, b.for_lend, b.for_sale, b.price, b.status"

//scanner lets scanbook work for both a single row and a row inside a loop
type scanner interface{
	Scan(dest ...any)error
}


// scanBook copies one database row into a Book struct(wil be created later in other file ).
func scanBook(s scanner) (Book, error) {
	var b Book
	err := s.Scan(&b.ID, &b.Title, &b.Author, &b.Genre, &b.Description,
		&b.OwnerID, &b.ForLend, &b.ForSale, &b.Price, &b.Status)
	return b, err
}


// readBooks loops over query results and returns a list of books.
// It also closes the rows when done.
func readBooks(rows *sql.Rows) []Book {
	defer rows.Close()
	list := []Book{} // empty list (so JSON shows [] not null)
	for rows.Next() {
		b, err := scanBook(rows)
		if err == nil {
			list = append(list, b)
		}
	}
	return list
}


// getBookByID finds one book. Returns an error if it doesn't exist.
func getBookByID(id int) (Book, error) {
	row := db.QueryRow("SELECT "+bookColumns+" FROM books b WHERE b.id = ?", id)
	return scanBook(row)
}


// validateBookInput returns an error message, or "" if everything is fine.
// bookinput is a structure declared in modelsf.go
func validateBookInput(in BookInput) string {
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Author) == "" {
		return "title and author are required"
	}
	if !in.ForLend && !in.ForSale {
		return "book must be for_lend, for_sale, or both"
	}
	if in.ForSale && in.Price <= 0 {
		return "a book for sale needs a price greater than 0"
	}
	return ""
}


//GET /books
func getAllBooksHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// We build the SQL step by step, adding a condition for every filter used.
	query := "SELECT " + bookColumns + " FROM books b WHERE 1=1"
	args := []any{}

	if search := strings.ToLower(strings.TrimSpace(q.Get("search"))); search != "" {
		query += " AND (LOWER(b.title) LIKE ? OR LOWER(b.author) LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	if genre := strings.ToLower(strings.TrimSpace(q.Get("genre"))); genre != "" {
		query += " AND LOWER(b.genre) = ?"
		args = append(args, genre)
	}
	if author := strings.ToLower(strings.TrimSpace(q.Get("author"))); author != "" {
		query += " AND LOWER(b.author) LIKE ?"
		args = append(args, "%"+author+"%")
	}
	if v := q.Get("min_price"); v != "" {
		minPrice, err := strconv.ParseFloat(v, 64)
		if err != nil {
			sendError(w, http.StatusBadRequest, "min_price must be a number")
			return
		}
		query += " AND b.for_sale = 1 AND b.price >= ?"
		args = append(args, minPrice)
	}
	if v := q.Get("max_price"); v != "" {
		maxPrice, err := strconv.ParseFloat(v, 64)
		if err != nil {
			sendError(w, http.StatusBadRequest, "max_price must be a number")
			return
		}
		query += " AND b.for_sale = 1 AND b.price <= ?"
		args = append(args, maxPrice)
	}
	if q.Get("available") == "true" {
		query += " AND b.status = 'available'"
	}
	switch q.Get("type") {
	case "lend":
		query += " AND b.for_lend = 1"
	case "sale":
		query += " AND b.for_sale = 1"
	}

	query += " ORDER BY b.id DESC" // newest first

	rows, err := db.Query(query, args...)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "could not load books")
		return
	}
	sendJSON(w, http.StatusOK, readBooks(rows))
}
