package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sacramento-finance/backend/internal/domain/notification"
)

type NotificationRepo struct {
	db *pgxpool.Pool
}

func NewNotificationRepo(db *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(ctx context.Context, n *notification.Notification) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO notifications (id, user_id, fund_id, type, title, body, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, n.ID, n.UserID, n.FundID, n.Type, n.Title, n.Body, n.IsRead, n.CreatedAt)
	return err
}

func (r *NotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*notification.Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, fund_id, type, title, body, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY is_read ASC, created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*notification.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *NotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE
	`, userID).Scan(&count)
	return count, err
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE
	`, userID)
	return err
}

type notifScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row notifScanner) (*notification.Notification, error) {
	n := &notification.Notification{}
	var createdAt time.Time
	err := row.Scan(&n.ID, &n.UserID, &n.FundID, &n.Type, &n.Title, &n.Body, &n.IsRead, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	n.CreatedAt = createdAt
	return n, nil
}
