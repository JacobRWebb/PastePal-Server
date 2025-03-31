package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"github.com/JacobRWebb/PastePal-Server/internal/models"
)

type FirebaseAuth struct {
	AuthClient *auth.Client
	Firestore  *firestore.Client
}

func NewFirebaseAuth(authClient *auth.Client, firestoreClient *firestore.Client) *FirebaseAuth {
	return &FirebaseAuth{
		AuthClient: authClient,
		Firestore:  firestoreClient,
	}
}

func (fa *FirebaseAuth) Register(w http.ResponseWriter, r *http.Request) {
	fmt.Println("\n==== REGISTER USER ROUTE ====")
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	
	fmt.Println("Raw request body:", string(bodyBytes))
	
	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	
	var regData models.RegistrationData
	if err := json.NewDecoder(r.Body).Decode(&regData); err != nil {
		fmt.Println("Error decoding JSON:", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	params := (&auth.UserToCreate{}).Email(regData.Email).Password(regData.PasswordHash)
	firebaseUser, err := fa.AuthClient.CreateUser(r.Context(), params)
	if err != nil {
		fmt.Println("Firebase error creating user:", err)
		http.Error(w, "Failed to create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	userData := map[string]interface{}{
		"email":                 regData.Email,
		"encrypted_symmetric_key": regData.EncryptedSymmetricKey,
		"created_at":           firestore.ServerTimestamp,
		"password_hash":        regData.PasswordHash,
	}
	
	fmt.Println("Storing user data in Firestore...")
	_, err = fa.Firestore.Collection("users").Doc(firebaseUser.UID).Set(r.Context(), userData)
	if err != nil {
		fmt.Println("Firestore error:", err)
		fa.AuthClient.DeleteUser(r.Context(), firebaseUser.UID)
		http.Error(w, "Failed to store user data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	token, err := fa.AuthClient.CustomToken(r.Context(), firebaseUser.UID)
	if err != nil {
		fmt.Println("Error creating custom token:", err)
		http.Error(w, "Failed to create token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

func (fa *FirebaseAuth) Login(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	var loginReq models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		fmt.Println("Error decoding login JSON:", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	firebaseUser, err := fa.AuthClient.GetUserByEmail(r.Context(), loginReq.Email)
	if err != nil {
		fmt.Println("Firebase error getting user:", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	userDoc, err := fa.Firestore.Collection("users").Doc(firebaseUser.UID).Get(r.Context())
	if err != nil {
		fmt.Println("Firestore error getting user data:", err)
		http.Error(w, "Failed to retrieve user data", http.StatusInternalServerError)
		return
	}

	userData := userDoc.Data()
	storedPasswordHash, ok := userData["password_hash"].(string)
	if !ok {
		fmt.Println("Error: password_hash not found in Firestore or not a string")
		http.Error(w, "User data incomplete", http.StatusInternalServerError)
		return
	}

	if loginReq.PasswordHash != storedPasswordHash {
		fmt.Println("Password hash mismatch, trying Firebase authentication...")
		fmt.Println("WARNING: Password hash mismatch. Firebase Auth doesn't provide a direct way to verify passwords.")
	}

	token, err := fa.AuthClient.CustomToken(r.Context(), firebaseUser.UID)
	if err != nil {
		fmt.Println("Firebase error creating token:", err)
		http.Error(w, "Failed to create token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	encryptedSymmetricKey, ok := userData["encrypted_symmetric_key"].(string)
	if !ok {
		fmt.Println("Error: encrypted_symmetric_key not found or not a string")
		http.Error(w, "User data incomplete", http.StatusInternalServerError)
		return
	}

	user := models.User{
		ID:        firebaseUser.UID,
		Email:     firebaseUser.Email,
		CreatedAt: time.Now(),
	}

	response := models.LoginResponse{
		User:                 user,
		AuthToken:            token,
		EncryptedSymmetricKey: encryptedSymmetricKey,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Authorization", "Bearer "+token)
	json.NewEncoder(w).Encode(response)
}

func (fa *FirebaseAuth) VerifyToken(ctx context.Context, token string) (*auth.Token, error) {
	return fa.AuthClient.VerifyIDToken(ctx, token)
}
