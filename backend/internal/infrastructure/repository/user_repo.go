package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sacramento-finance/backend/internal/domain/user"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *user.User) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO users (
			id, document_type, document_number, email, phone, full_name,
			password_hash, is_verified, verification_status, verification_ref,
			verified_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`,
		u.ID, u.DocumentType, u.DocumentNumber, u.Email, u.Phone, u.FullName,
		u.PasswordHash, u.IsVerified, u.VerificationStatus, u.VerificationRef,
		u.VerifiedAt, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			// Inspect constraint name to return the right error
			var pgErr *pgconn.PgError
			errors.As(err, &pgErr)
			if pgErr.ConstraintName == "users_document_unique" {
				return apperror.ErrDuplicateCedula
			}
			return apperror.ErrDuplicateEmail
		}
		return err
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return r.scanOne(ctx, `
		SELECT id, document_type, document_number, email, phone, full_name,
		       password_hash, is_verified, verification_status, verification_ref,
		       verified_at, created_at, updated_at
		FROM users WHERE id = $1
	`, id)
}

func (r *UserRepo) GetByDocument(ctx context.Context, docType user.DocumentType, docNumber string) (*user.User, error) {
	return r.scanOne(ctx, `
		SELECT id, document_type, document_number, email, phone, full_name,
		       password_hash, is_verified, verification_status, verification_ref,
		       verified_at, created_at, updated_at
		FROM users WHERE document_type = $1 AND document_number = $2
	`, docType, docNumber)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.scanOne(ctx, `
		SELECT id, document_type, document_number, email, phone, full_name,
		       password_hash, is_verified, verification_status, verification_ref,
		       verified_at, created_at, updated_at
		FROM users WHERE email = $1
	`, email)
}

func (r *UserRepo) Update(ctx context.Context, u *user.User) error {
	u.UpdatedAt = time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET phone=$1, full_name=$2, is_verified=$3,
		    verification_status=$4, verification_ref=$5, verified_at=$6,
		    updated_at=$7
		WHERE id=$8
	`, u.Phone, u.FullName, u.IsVerified,
		u.VerificationStatus, u.VerificationRef, u.VerifiedAt,
		u.UpdatedAt, u.ID)
	return err
}

func (r *UserRepo) scanOne(ctx context.Context, query string, args ...any) (*user.User, error) {
	row := r.db.QueryRow(ctx, query, args...)
	u := &user.User{}
	err := row.Scan(
		&u.ID, &u.DocumentType, &u.DocumentNumber, &u.Email, &u.Phone, &u.FullName,
		&u.PasswordHash, &u.IsVerified, &u.VerificationStatus, &u.VerificationRef,
		&u.VerifiedAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
