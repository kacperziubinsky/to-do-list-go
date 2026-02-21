package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"moj_pierwszy_projekt/db"
	"moj_pierwszy_projekt/middleware"
	"moj_pierwszy_projekt/models"

	"golang.org/x/crypto/bcrypt"
)

func generateToken() string {
	return fmt.Sprintf("token_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	var exists int
	err := db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", req.Username).Scan(&exists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if exists > 0 {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Bcrypt error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	result, err := db.DB.Exec("INSERT INTO users (username, password) VALUES (?, ?)", req.Username, string(hashedPassword))
	if err != nil {
		log.Printf("User insert error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	userID, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "User registered successfully",
		"user_id":  userID,
		"username": req.Username,
	})
}

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	var user models.User
	var hashedPassword string
	err := db.DB.QueryRow("SELECT id, username, password FROM users WHERE username = ?", req.Username).
		Scan(&user.ID, &user.Username, &hashedPassword)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token := generateToken()
	middleware.SetSession(token, user.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.LoginResponse{
		Token:    token,
		Username: user.Username,
		UserID:   user.ID,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.Header.Get("Authorization")
	token = trimBearer(token)
	middleware.DeleteSession(token)

	w.WriteHeader(http.StatusNoContent)
}

func trimBearer(token string) string {
	if len(token) > 7 && token[:7] == "Bearer " {
		return token[7:]
	}
	return token
}