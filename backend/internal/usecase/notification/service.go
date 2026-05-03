package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/pkg/idgen"
)

// emailableTypes lists notification types that should also trigger an email.
var emailableTypes = map[notification.Type]bool{
	notification.TypePaymentOverdue:   true,
	notification.TypePayoutReceived:   true,
	notification.TypeWithdrawal:       true,
	notification.TypeGoalReached:      true,
	notification.TypeProposalCreated:  true,
	notification.TypeProposalResolved: true,
}

// Service wraps the repository and exposes typed helpers for each event.
// All methods are safe to call in a goroutine (fire-and-forget).
type Service struct {
	repo        notification.Repository
	emailSender notification.EmailSender // nil when email is disabled
	userEmails  UserEmailLookup          // nil when email is disabled
}

// UserEmailLookup resolves a userID to an email address for outbound emails.
type UserEmailLookup interface {
	GetEmailByUserID(ctx context.Context, userID uuid.UUID) (string, error)
}

func NewService(repo notification.Repository) *Service {
	return &Service{repo: repo}
}

// WithEmail attaches an email sender and user lookup so critical events also send email.
func (s *Service) WithEmail(sender notification.EmailSender, lookup UserEmailLookup) *Service {
	s.emailSender = sender
	s.userEmails = lookup
	return s
}

// NotifyOne creates a single in-app notification and optionally sends an email.
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

	if s.emailSender != nil && s.userEmails != nil && emailableTypes[notifType] {
		if email, err := s.userEmails.GetEmailByUserID(ctx, userID); err == nil && email != "" {
			_ = s.emailSender.Send(ctx, email, title, body)
		}
	}
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
