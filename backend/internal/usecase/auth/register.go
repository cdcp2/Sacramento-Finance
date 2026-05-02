package auth

import (
	"context"
	"time"

	"github.com/sacramento-finance/backend/internal/domain/user"
	"github.com/sacramento-finance/backend/pkg/apperror"
	"github.com/sacramento-finance/backend/pkg/idgen"
	"golang.org/x/crypto/bcrypt"
)

type RegisterInput struct {
	DocumentType   user.DocumentType
	DocumentNumber string
	Email          string
	Phone          string
	FullName       string
	Password       string
}

type RegisterUseCase struct {
	users user.Repository
}

func NewRegisterUseCase(users user.Repository) *RegisterUseCase {
	return &RegisterUseCase{users: users}
}

func (uc *RegisterUseCase) Execute(ctx context.Context, in RegisterInput) (*user.UserPublic, error) {
	if existing, _ := uc.users.GetByDocument(ctx, in.DocumentType, in.DocumentNumber); existing != nil {
		return nil, apperror.ErrDuplicateCedula
	}

	if existing, _ := uc.users.GetByEmail(ctx, in.Email); existing != nil {
		return nil, apperror.ErrDuplicateEmail
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.ErrInternal
	}

	now := time.Now().UTC()
	u := &user.User{
		ID:                 idgen.New(),
		DocumentType:       in.DocumentType,
		DocumentNumber:     in.DocumentNumber,
		Email:              in.Email,
		Phone:              in.Phone,
		FullName:           in.FullName,
		PasswordHash:       string(hash),
		IsVerified:         false,
		VerificationStatus: user.VerificationNone,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := uc.users.Create(ctx, u); err != nil {
		return nil, err
	}

	pub := u.Sanitized()
	return &pub, nil
}
