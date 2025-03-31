package config

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// Config holds the application configuration
type Config struct {
	FirebaseApp  *firebase.App
	FirebaseAuth *auth.Client
	Firestore    *firestore.Client
	Port         string
}

// findAvailablePort tries to find an available port starting from the given port
func findAvailablePort(startPort int) int {
	for port := startPort; port < startPort+100; port++ {
		address := ":"+strconv.Itoa(port)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			listener.Close()
			return port
		}
	}
	return startPort // fallback to original port if no available ports found
}

// New creates a new application configuration
func New() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Try to find an available port if the specified port is in use
	portInt, _ := strconv.Atoi(port)
	availablePort := findAvailablePort(portInt)
	port = strconv.Itoa(availablePort)

	firebaseCredentialsPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	if firebaseCredentialsPath == "" {
		log.Fatal("FIREBASE_CREDENTIALS_PATH environment variable is required")
	}

	opt := option.WithCredentialsFile(firebaseCredentialsPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v\n", err)
	}

	auth, err := app.Auth(context.Background())
	if err != nil {
		log.Fatalf("Error initializing Firebase auth: %v\n", err)
	}

	// Initialize Firestore
	firestoreClient, err := app.Firestore(context.Background())
	if err != nil {
		log.Fatalf("Error initializing Firestore: %v\n", err)
	}

	return &Config{
		FirebaseApp:  app,
		FirebaseAuth: auth,
		Firestore:    firestoreClient,
		Port:         port,
	}
}
