package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"pc4/auth"
	"pc4/database"
	"pc4/tools"
)

func LoginUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tools.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	user, err := database.FindUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "user wasn't found", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPassword(user.Password, req.Password); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.CreateJWT(user.Email)
	if err != nil {
		http.Error(w, "could not create token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})

}

func RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tools.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Registering userId=%d email=%s\n", req.UserID, req.Email)

	hashedPw, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "invalid password", http.StatusBadRequest)
		return
	}

	req.Password = hashedPw

	exists, err := database.UserExists(req.UserID)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "user not created", http.StatusNotFound)
		return
	}

	if err = database.CreateUserWeb(r.Context(), req); err != nil {
		http.Error(w, "user couldn't be created", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "user %s registered", req.Email)
}
