package notification_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	domainnotif "github.com/sacramento-finance/backend/internal/domain/notification"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	ucnotif "github.com/sacramento-finance/backend/internal/usecase/notification"
)

// ─── mock repo ────────────────────────────────────────────────────────────────

type mockRepo struct {
	created    []*domainnotif.Notification
	createErr  error
	listed     []*domainnotif.Notification
	unread     int
	markReadID uuid.UUID
	markReadErr error
	markAllErr error
}

func (m *mockRepo) Create(_ context.Context, n *domainnotif.Notification) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, n)
	return nil
}
func (m *mockRepo) ListByUser(_ context.Context, _ uuid.UUID, _, _ int) ([]*domainnotif.Notification, error) {
	return m.listed, nil
}
func (m *mockRepo) CountUnread(_ context.Context, _ uuid.UUID) (int, error) {
	return m.unread, nil
}
func (m *mockRepo) MarkRead(_ context.Context, id, _ uuid.UUID) error {
	m.markReadID = id
	return m.markReadErr
}
func (m *mockRepo) MarkAllRead(_ context.Context, _ uuid.UUID) error {
	return m.markAllErr
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func activeMember(fundID uuid.UUID) *fund.FundMember {
	return &fund.FundMember{
		ID:     uuid.New(),
		FundID: fundID,
		UserID: uuid.New(),
		Status: fund.MemberStatusActive,
	}
}

func inactiveMember(fundID uuid.UUID) *fund.FundMember {
	return &fund.FundMember{
		ID:     uuid.New(),
		FundID: fundID,
		UserID: uuid.New(),
		Status: fund.MemberStatusLeft,
	}
}

// ─── NotifyOne ────────────────────────────────────────────────────────────────

func TestNotifyOne_CreatesNotificationWithCorrectFields(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)

	userID := uuid.New()
	fundID := uuid.New()
	svc.NotifyOne(context.Background(), userID, &fundID,
		domainnotif.TypePaymentConfirmed, "Aporte confirmado", "Tu aporte fue confirmado.")

	if len(repo.created) != 1 {
		t.Fatalf("expected 1 notification created, got %d", len(repo.created))
	}
	n := repo.created[0]

	if n.UserID != userID {
		t.Errorf("UserID = %s, want %s", n.UserID, userID)
	}
	if n.FundID == nil || *n.FundID != fundID {
		t.Errorf("FundID = %v, want %s", n.FundID, fundID)
	}
	if n.Type != domainnotif.TypePaymentConfirmed {
		t.Errorf("Type = %s, want payment_confirmed", n.Type)
	}
	if n.Title != "Aporte confirmado" {
		t.Errorf("Title = %q, want %q", n.Title, "Aporte confirmado")
	}
	if n.Body != "Tu aporte fue confirmado." {
		t.Errorf("Body = %q", n.Body)
	}
	if n.IsRead {
		t.Error("new notification should not be read")
	}
	if n.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if n.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestNotifyOne_NilFundID_StoresNilFundID(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)

	svc.NotifyOne(context.Background(), uuid.New(), nil,
		domainnotif.TypeMemberJoined, "Bienvenido", "")

	if len(repo.created) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(repo.created))
	}
	if repo.created[0].FundID != nil {
		t.Errorf("expected nil FundID, got %v", repo.created[0].FundID)
	}
}

func TestNotifyOne_RepoError_IsSilenced(t *testing.T) {
	repo := &mockRepo{createErr: errors.New("db down")}
	svc := ucnotif.NewService(repo)

	// Should not panic or return error
	svc.NotifyOne(context.Background(), uuid.New(), nil,
		domainnotif.TypePaymentConfirmed, "t", "b")
}

// ─── NotifyAll ────────────────────────────────────────────────────────────────

func TestNotifyAll_NotifiesAllActiveMembers(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)
	fundID := uuid.New()

	members := []*fund.FundMember{
		activeMember(fundID),
		activeMember(fundID),
		activeMember(fundID),
	}

	svc.NotifyAll(context.Background(), members, nil, &fundID,
		domainnotif.TypeRoundClosed, "Ronda cerrada", "La ronda fue cerrada.")

	if len(repo.created) != 3 {
		t.Errorf("expected 3 notifications, got %d", len(repo.created))
	}
}

func TestNotifyAll_SkipsInactiveMembers(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)
	fundID := uuid.New()

	members := []*fund.FundMember{
		activeMember(fundID),
		inactiveMember(fundID),
		activeMember(fundID),
	}

	svc.NotifyAll(context.Background(), members, nil, &fundID,
		domainnotif.TypeRoundClosed, "Ronda cerrada", "")

	if len(repo.created) != 2 {
		t.Errorf("expected 2 notifications (active only), got %d", len(repo.created))
	}
}

func TestNotifyAll_SkipsExcludedUser(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)
	fundID := uuid.New()

	actor := activeMember(fundID)
	others := []*fund.FundMember{
		activeMember(fundID),
		activeMember(fundID),
		actor,
	}

	svc.NotifyAll(context.Background(), others, &actor.UserID, &fundID,
		domainnotif.TypeProposalCreated, "Nueva propuesta", "")

	if len(repo.created) != 2 {
		t.Errorf("expected 2 notifications (excluding actor), got %d", len(repo.created))
	}
	for _, n := range repo.created {
		if n.UserID == actor.UserID {
			t.Error("actor should not receive their own notification")
		}
	}
}

func TestNotifyAll_EmptyMembers_CreatesNothing(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)
	fundID := uuid.New()

	svc.NotifyAll(context.Background(), nil, nil, &fundID,
		domainnotif.TypeRoundClosed, "t", "b")

	if len(repo.created) != 0 {
		t.Errorf("expected 0 notifications for empty members, got %d", len(repo.created))
	}
}

func TestNotifyAll_AllInactive_CreatesNothing(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)
	fundID := uuid.New()

	members := []*fund.FundMember{
		inactiveMember(fundID),
		inactiveMember(fundID),
	}

	svc.NotifyAll(context.Background(), members, nil, &fundID,
		domainnotif.TypeRoundClosed, "t", "b")

	if len(repo.created) != 0 {
		t.Errorf("expected 0 notifications for all-inactive members, got %d", len(repo.created))
	}
}

func TestNotifyAll_ExcludeUserNotInList_StillNotifiesAll(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)
	fundID := uuid.New()

	members := []*fund.FundMember{
		activeMember(fundID),
		activeMember(fundID),
	}
	outsider := uuid.New()

	svc.NotifyAll(context.Background(), members, &outsider, &fundID,
		domainnotif.TypeMemberJoined, "Nuevo miembro", "")

	// outsider is not in the list, so all active members should be notified
	if len(repo.created) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(repo.created))
	}
}

func TestNotifyAll_EachNotificationHasCorrectFundID(t *testing.T) {
	repo := &mockRepo{}
	svc := ucnotif.NewService(repo)
	fundID := uuid.New()

	members := []*fund.FundMember{activeMember(fundID), activeMember(fundID)}
	svc.NotifyAll(context.Background(), members, nil, &fundID,
		domainnotif.TypeInterestAccrued, "Interés aplicado", "")

	for i, n := range repo.created {
		if n.FundID == nil || *n.FundID != fundID {
			t.Errorf("notification[%d] FundID = %v, want %s", i, n.FundID, fundID)
		}
	}
}
