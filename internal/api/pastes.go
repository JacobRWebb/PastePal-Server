package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/JacobRWebb/PastePal-Server/internal/middleware"
	"github.com/JacobRWebb/PastePal-Server/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// PasteHandler handles paste-related API endpoints
type PasteHandler struct {
	Firestore *firestore.Client
}

// NewPasteHandler creates a new paste handler
func NewPasteHandler(firestoreClient *firestore.Client) *PasteHandler {
	return &PasteHandler{
		Firestore: firestoreClient,
	}
}

// CreatePaste handles the creation of a new paste
func (h *PasteHandler) CreatePaste(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.CreatePasteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate content type
	if req.ContentType != models.ContentTypeText && req.ContentType != models.ContentTypeImage {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	// For images, validate that the content is valid base64
	if req.ContentType == models.ContentTypeImage {
		// Check if the content is a data URL (e.g., data:image/png;base64,...)
		if strings.HasPrefix(req.Content, "data:") {
			// Extract the MIME type and base64 content
			parts := strings.Split(req.Content, ",")
			if len(parts) != 2 {
				http.Error(w, "Invalid image format", http.StatusBadRequest)
				return
			}
			
			// Extract MIME type from the data URL
			mimeInfo := strings.Split(parts[0], ":")
			if len(mimeInfo) != 2 {
				http.Error(w, "Invalid image format", http.StatusBadRequest)
				return
			}
			
			mimeType := strings.Split(mimeInfo[1], ";")
			if len(mimeType) != 2 || mimeType[1] != "base64" {
				http.Error(w, "Only base64 encoded images are supported", http.StatusBadRequest)
				return
			}
			
			// Set the MIME type
			req.MimeType = mimeType[0]
			
			// Validate the base64 content
			_, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				http.Error(w, "Invalid base64 encoding: "+err.Error(), http.StatusBadRequest)
				return
			}
			
			// Store only the base64 part
			req.Content = parts[1]
		} else {
			// If it's not a data URL, validate it as raw base64
			_, err := base64.StdEncoding.DecodeString(req.Content)
			if err != nil {
				http.Error(w, "Invalid base64 encoding: "+err.Error(), http.StatusBadRequest)
				return
			}
			
			// If no MIME type was provided, use a default
			if req.MimeType == "" {
				req.MimeType = "image/png"
			}
		}
	}

	// Create a new paste
	now := time.Now()
	pasteID := uuid.New().String()
	
	paste := &models.Paste{
		ID:          pasteID,
		UserID:      userID,
		Title:       req.Title,
		Content:     req.Content,
		ContentType: req.ContentType,
		MimeType:    req.MimeType,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsPublic:    req.IsPublic,
	}

	// Store the paste in Firestore
	_, err := h.Firestore.Collection("pastes").Doc(pasteID).Set(r.Context(), paste)
	if err != nil {
		fmt.Println("Error storing paste in Firestore:", err)
		http.Error(w, "Failed to store paste: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the created paste
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(paste)
}

// GetPaste retrieves a paste by ID
func (h *PasteHandler) GetPaste(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pasteID := vars["id"]

	// Get the paste from Firestore
	doc, err := h.Firestore.Collection("pastes").Doc(pasteID).Get(r.Context())
	if err != nil {
		fmt.Println("Error retrieving paste from Firestore:", err)
		http.Error(w, "Paste not found", http.StatusNotFound)
		return
	}

	// Convert Firestore document to Paste struct
	var paste models.Paste
	if err := doc.DataTo(&paste); err != nil {
		fmt.Println("Error converting Firestore document to Paste:", err)
		http.Error(w, "Error processing paste data", http.StatusInternalServerError)
		return
	}

	// Check if the paste is public or if the user is the owner
	if !paste.IsPublic {
		userID, ok := middleware.GetUserID(r.Context())
		if !ok || userID != paste.UserID {
			http.Error(w, "Unauthorized to access this paste", http.StatusForbidden)
			return
		}
	}

	// For images, reconstruct the data URL if needed
	if paste.ContentType == models.ContentTypeImage && !strings.HasPrefix(paste.Content, "data:") {
		dataURL := fmt.Sprintf("data:%s;base64,%s", paste.MimeType, paste.Content)
		paste.Content = dataURL
	}

	// Return the paste
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(paste)
}

// GetUserPastes retrieves all pastes for the authenticated user
func (h *PasteHandler) GetUserPastes(w http.ResponseWriter, r *http.Request) {
	fmt.Println("\n==== GET USER PASTES ROUTE ====")
	// Get user ID from context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Query Firestore for pastes by user ID
	iter := h.Firestore.Collection("pastes").Where("user_id", "==", userID).OrderBy("created_at", firestore.Desc).Documents(r.Context())
	defer iter.Stop()

	// Collect all pastes
	var pastes []*models.Paste
	for {
		doc, err := iter.Next()
		if err != nil {
			break // No more documents
		}

		var paste models.Paste
		if err := doc.DataTo(&paste); err != nil {
			fmt.Println("Error converting Firestore document to Paste:", err)
			continue // Skip this paste and continue
		}

		// For images, reconstruct the data URL if needed
		if paste.ContentType == models.ContentTypeImage && !strings.HasPrefix(paste.Content, "data:") {
			dataURL := fmt.Sprintf("data:%s;base64,%s", paste.MimeType, paste.Content)
			paste.Content = dataURL
		}

		pastes = append(pastes, &paste)
	}

	// Return the user's pastes
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pastes)
}
