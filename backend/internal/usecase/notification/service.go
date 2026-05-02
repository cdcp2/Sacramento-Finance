package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/pkg/idgen"
)

// Service wraps the repository and exposes typed helpers for each event.
// All methods are safe to call in a goroutine (fire-and-forget).
type Service struct {
	repo notification.Repository
}

func NewService(repo notification.Repository) *Service {
	return &Service{repo: repo}
}

// NotifyOne creates a single notification for a user.
func (s *Service) NotifyOne(
	ctx context.Context,
	userID uuid.UUID,
	fundID *uuid.UUID,
	notifType notification.Type,
	title, body string,
) {
	n := &notification.Notification{
		ID:        idgen.New(),
		UserID:    userID,
		FundID:    fundID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}
	_ = s.repo.Create(ctx, n)
}

// NotifyAll creates one notification per active fund member, skipping excludeUserID if provided.
func (s *Service) NotifyAll(
	ctx context.Context,
	members []*fund.FundMember,
	excludeUserID *uuid.UUID,
	fundID *uuid.UUID,
	notifType notification.Type,
	title, body string,
) {
	for _, m := range members {
		if m.Status != fund.MemberStatusActive {
			continue
		}
		if excludeUserID != nil && m.UserID == *excludeUserID {
			continue
		}
		s.NotifyOne(ctx, m.UserID, fundID, notifType, title, body)
	}
}
