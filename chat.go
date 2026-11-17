package main 



import (
	"net/http"
	"strconv"
	"strings"
)

type Message struct {
	ID 			int 	`json:"id"`
	BookID      int 	`json:"book_id"`
	SenderID	int		`json:"sender_id"`
	SenderName  string  `json:"sender_name"`
	Text     	string 	`json:"text"`
	CreatedAt	string 	`json:"created_at"`
}

type Chat struct {
	BookID       int    `json:"book_id"`
	BookTitle    string `json:"book_title"`
	CustomerID   int    `json:"customer_id"`
	CustomerName string `json:"customer_name"`
	IAmOwner     bool   `json:"i_am_owner"`
}

// POST /books/{id}/messages  (login needed)
// Customer body: {"text": "Hi, can I borrow this book?"}
// Owner body:    {"text": "Yes!", "customer_id": 7}
func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getLoggedInUserID(r)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Please log in to send a message")
		return
	}
	bookID, err := getIDFromURL(r, "id")
	if err != nil {
		sendError(w, http.StatusBadRequest, "book id must be a number")
		return
	}
	book, err := getBookByID(bookID)
	if err != nil {
		sendError(w, http.StatusNotFound, "book not found")
		return
	}

	var input struct {
		Text       string `json:"text"`
		CustomerID int    `json:"customer_id"`
	}
	if err := readJSON(r, &input); err != nil {
		sendError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	input.Text = strings.TrimSpace(input.Text)
	if input.Text == "" {
		sendError(w, http.StatusBadRequest, "message cannot be empty")
		return
	}
	if len(input.Text) > 1000 {
		sendError(w, http.StatusBadRequest, "message is too long (max 1000 characters)")
		return
	}

	// Who is the customer in this chat?
	customerID := userID // if I am a customer, the chat is mine
	if userID == book.OwnerID {
		// I am the owner, so I must say which customer I am answering,
		// and that customer must have written to me first.
		customerID = input.CustomerID
		var count int
		db.QueryRow("SELECT COUNT(*) FROM messages WHERE book_id = ? AND customer_id = ?",
			bookID, customerID).Scan(&count)
		if count == 0 {
			sendError(w, http.StatusForbidden, "that person has not written to you about this book")
			return
		}
	}

	_, err = db.Exec("INSERT INTO messages (book_id, customer_id, sender_id, text, created_at) VALUES (?, ?, ?, ?, ?)",
		bookID, customerID, userID, input.Text, now())
	if err != nil {
		sendError(w, http.StatusInternalServerError, "could not send message")
		return
	}
	sendJSON(w, http.StatusCreated, map[string]string{"message": "sent"})
}

func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getLoggedInUserID(r)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Please log in to see messages")
		return
	}
	bookID, err := getIDFromURL(r, "id")
	if err != nil {
		sendError(w, http.StatusBadRequest, "book id must be a number")
		return
	}
	book, err := getBookByID(bookID)
	if err != nil {
		sendError(w, http.StatusNotFound, "book not found")
		return
	}

	customerID := userID
	if userID == book.OwnerID {
		customerID, err = strconv.Atoi(r.URL.Query().Get("customer_id"))
		if err != nil {
			sendError(w, http.StatusBadRequest, "customer_id is required")
			return
		}
	}

	rows, err := db.Query(`SELECT m.id, m.book_id, m.sender_id, u.name, m.text, m.created_at
		FROM messages m JOIN users u ON u.id = m.sender_id
		WHERE m.book_id = ? AND m.customer_id = ? ORDER BY m.id`, bookID, customerID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "could not load messages")
		return
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		rows.Scan(&m.ID, &m.BookID, &m.SenderID, &m.SenderName, &m.Text, &m.CreatedAt)
		messages = append(messages, m)
	}
	sendJSON(w, http.StatusOK, messages)
}

func myChatsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := getLoggedInUserID(r)
	if err != nil {
		sendError(w, http.StatusUnauthorized, "Please log in to see your chats")
		return
	}

	rows, err := db.Query(`SELECT DISTINCT m.book_id, b.title, b.owner_id, m.customer_id, u.name
		FROM messages m
		JOIN books b ON b.id = m.book_id
		JOIN users u ON u.id = m.customer_id
		WHERE b.owner_id = ? OR m.customer_id = ?
		ORDER BY m.book_id DESC`, userID, userID)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "could not load chats")
		return
	}
	defer rows.Close()

	chats := []Chat{}
	for rows.Next() {
		var c Chat
		var ownerID int
		rows.Scan(&c.BookID, &c.BookTitle, &ownerID, &c.CustomerID, &c.CustomerName)
		c.IAmOwner = ownerID == userID
		chats = append(chats, c)
	}
	sendJSON(w, http.StatusOK, chats)
}


