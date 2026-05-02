package user

import (
	"time"

	"github.com/google/uuid"
)

type DocumentType string

const (
	DocumentCedulaCiudadania  DocumentType = "cedula_ciudadania"
	DocumentCedulaExtranjeria DocumentType = "cedula_extranjeria"
	DocumentPasaporte         DocumentType = "pasaporte"
)

type VerificationStatus string

const (
	VerificationNone     VerificationStatus = "none"
	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

type User struct {
	ID                 uuid.UUID
	DocumentType       DocumentType
	DocumentNumber     string
	Email              string
	Phone              string
	FullName           string
	PasswordHash       string
	IsVerified         bool
	VerificationStatus VerificationStatus
	VerificationRef    *string
	VerifiedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (u *User) Sanitized() UserPublic {
	return UserPublic{
		ID:                 u.ID,
		DocumentType:       u.DocumentType,
		DocumentNumber:     u.DocumentNumber,
		Email:              u.Email,
		Phone:              u.Phone,
		FullName:           u.FullName,
		IsVerified:         u.IsVerified,
		VerificationStatus: u.VerificationStatus,
		CreatedAt:          u.CreatedAt,
	}
}

type UserPublic struct {
	ID                 uuid.UUID          `json:"id"`
	DocumentType       DocumentType       `json:"document_type"`
	DocumentNumber     string             `json:"document_number"`
	Email              string             `json:"email"`
	Phone              string             `json:"phone"`
	FullName           string             `json:"full_name"`
	IsVerified         bool               `json:"is_verified"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	CreatedAt          time.Time          `json:"created_at"`
}
