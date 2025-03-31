package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/JacobRWebb/PastePal-Server/internal/api"
	"github.com/JacobRWebb/PastePal-Server/internal/auth"
	"github.com/JacobRWebb/PastePal-Server/internal/config"
	"github.com/JacobRWebb/PastePal-Server/internal/middleware"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	}

	// Initialize configuration
	cfg := config.New()

	// Ensure Firestore client is closed on shutdown
	defer cfg.Firestore.Close()

	// Initialize Firebase auth handler
	firebaseAuth := auth.NewFirebaseAuth(cfg.FirebaseAuth, cfg.Firestore)

	// Initialize paste handler with Firestore client
	pasteHandler := api.NewPasteHandler(cfg.Firestore)

	// Create router
	r := mux.NewRouter()

	// Auth routes
	r.HandleFunc("/api/auth/register", firebaseAuth.Register).Methods("POST")
	r.HandleFunc("/api/auth/login", firebaseAuth.Login).Methods("POST")

	// Create a subrouter with auth middleware for protected routes
	protectedRouter := r.PathPrefix("/api").Subrouter()
	protectedRouter.Use(middleware.AuthMiddleware(cfg.FirebaseAuth))

	// Paste routes
	protectedRouter.HandleFunc("/pastes", pasteHandler.CreatePaste).Methods("POST")
	protectedRouter.HandleFunc("/pastes", pasteHandler.GetUserPastes).Methods("GET")
	r.HandleFunc("/api/pastes/{id}", pasteHandler.GetPaste).Methods("GET") // Public route with auth check inside

	// Start server
	port := cfg.Port
	log.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}
