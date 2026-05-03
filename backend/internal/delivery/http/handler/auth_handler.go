package handler

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/sacramento-finance/backend/internal/domain/user"
	"github.com/sacramento-finance/backend/internal/usecase/auth"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

type AuthHandler struct {
	register *auth.RegisterUseCase
	login    *auth.LoginUseCase
	refresh  *auth.RefreshUseCase
}

func NewAuthHandler(register *auth.RegisterUseCase, login *auth.LoginUseCase, refresh *auth.RefreshUseCase) *AuthHandler {
	return &AuthHandler{register: register, login: login, refresh: refresh}
}

type registerRequest struct {
	DocumentType   string `json:"document_type" binding:"required,oneof=cedula_ciudadania cedula_extranjeria pasaporte"`
	DocumentNumber string `json:"document_number" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Phone          string `json:"phone" binding:"required,numeric,min=7,max=15"`
	FullName       string `json:"full_name" binding:"required,min=3"`
	Password       string `json:"password" binding:"required,min=8"`
}

type loginRequest struct {
	DocumentType   string `json:"document_type"` // optional — if provided, used with document_number
	DocumentNumber string `json:"document_number"`
	Email          string `json:"email"`
	Password       string `json:"password" binding:"required"`
}

var (
	reOnlyDigits    = regexp.MustCompile(`^\d+$`)
	reAlphanumeric  = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	reLeadingZero   = regexp.MustCompile(`^0`)
)

// validateDocumentNumber returns an error message if the number is invalid for its type.
func validateDocumentNumber(docType, docNumber string) string {
	switch docType {
	case string(user.DocumentCedulaCiudadania):
		if !reOnlyDigits.MatchString(docNumber) {
			return "cedula_ciudadania must contain only digits"
		}
		if reLeadingZero.MatchString(docNumber) {
			return "cedula_ciudadania must not start with zero"
		}
		if len(docNumber) < 3 || len(docNumber) > 10 {
			return "cedula_ciudadania must be between 3 and 10 digits"
		}
	case string(user.DocumentCedulaExtranjeria):
		if !reAlphanumeric.MatchString(docNumber) {
			return "cedula_extranjeria must contain only letters and digits"
		}
		if len(docNumber) < 4 || len(docNumber) > 15 {
			return "cedula_extranjeria must be between 4 and 15 characters"
		}
	case string(user.DocumentPasaporte):
		if !reAlphanumeric.MatchString(docNumber) {
			return "pasaporte must contain only letters and digits"
		}
		if len(docNumber) < 5 || len(docNumber) > 20 {
			return "pasaporte must be between 5 and 20 characters"
		}
	}
	return ""
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	if msg := validateDocumentNumber(req.DocumentType, req.DocumentNumber); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_DOCUMENT_NUMBER", "message": msg}})
		return
	}

	result, err := h.register.Execute(c.Request.Context(), auth.RegisterInput{
		DocumentType:   user.DocumentType(req.DocumentType),
		DocumentNumber: req.DocumentNumber,
		Email:          req.Email,
		Phone:          req.Phone,
		FullName:       req.FullName,
		Password:       req.Password,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": result})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	// Must provide either email or (document_type + document_number)
	if req.Email == "" && (req.DocumentType == "" || req.DocumentNumber == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "provide either email or document_type + document_number",
		}})
		return
	}

	tokens, err := h.login.Execute(c.Request.Context(), auth.LoginInput{
		DocumentType:   user.DocumentType(req.DocumentType),
		DocumentNumber: req.DocumentNumber,
		Email:          req.Email,
		Password:       req.Password,
	})
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tokens})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	tokens, err := h.refresh.Execute(req.RefreshToken)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tokens})
}

func respondError(c *gin.Context, err error) {
	if appErr, ok := apperror.As(err); ok {
		c.JSON(appErr.Status, gin.H{"error": appErr})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": apperror.ErrInternal})
}
