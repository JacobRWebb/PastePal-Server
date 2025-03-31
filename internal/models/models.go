package models

import (
	"time"
)

// User represents a registered user in the system
type User struct {
	ID        string    `json:"id" firestore:"id"`
	Email     string    `json:"email" firestore:"email"`
	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
}

// RegistrationData contains the data needed to register a new user
type RegistrationData struct {
	Email                string `json:"email"`
	PasswordHash         string `json:"password_hash"`
	EncryptedSymmetricKey string `json:"encrypted_symmetric_key"`
}

// LoginRequest contains the credentials for user login
type LoginRequest struct {
	Email       string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

// LoginResponse contains the response data after successful login
type LoginResponse struct {
	User      User   `json:"user"`
	AuthToken string `json:"auth_token"`
	EncryptedSymmetricKey string `json:"encrypted_symmetric_key,omitempty"`
}

// ContentType defines the type of content in a paste
type ContentType string

// Content type constants
const (
	ContentTypeText  ContentType = "text"
	ContentTypeImage ContentType = "image"
)

// Paste represents a stored paste
type Paste struct {
	ID          string      `json:"id" firestore:"id"`
	UserID      string      `json:"user_id" firestore:"user_id"`
	Title       string      `json:"title" firestore:"title"`
	Content     string      `json:"content" firestore:"content"`
	ContentType ContentType `json:"content_type" firestore:"content_type"`
	MimeType    string      `json:"mime_type,omitempty" firestore:"mime_type,omitempty"`
	CreatedAt   time.Time   `json:"created_at" firestore:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" firestore:"updated_at"`
	IsPublic    bool        `json:"is_public" firestore:"is_public"`
}

// CreatePasteRequest contains the data needed to create a new paste
type CreatePasteRequest struct {
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	ContentType ContentType `json:"content_type"`
	MimeType    string      `json:"mime_type,omitempty"`
	IsPublic    bool        `json:"is_public"`
}
