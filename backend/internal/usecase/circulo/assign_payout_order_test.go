package circulo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/usecase/circulo"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

func activeMember(fundID uuid.UUID) *fund.FundMember {
	return &fund.FundMember{
		ID:     uuid.New(),
		FundID: fundID,
		UserID: uuid.New(),
		Status: fund.MemberStatusActive,
	}
}

func TestAssignPayoutOrder_Manual_AssignsProvidedOrders(t *testing.T) {
	f := activeCirculoFund(3)
	memberA := activeMember(f.ID)
	memberB := activeMember(f.ID)
	memberC := activeMember(f.ID)
	memberRepo := &mockMemberRepo{members: []*fund.FundMember{memberA, memberB, memberC}}
	uc := circulo.NewAssignPayoutOrderUseCase(&mockCirculoRepo{}, memberRepo)

	err := uc.Execute(context.Background(), f, []circulo.MemberOrder{
		{MemberID: memberA.ID.String(), Order: 2},
		{MemberID: memberB.ID.String(), Order: 1},
		{MemberID: memberC.ID.String(), Order: 3},
	}, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertPayoutOrder(t, memberA, 2)
	assertPayoutOrder(t, memberB, 1)
	assertPayoutOrder(t, memberC, 3)
	if len(memberRepo.updated) != 3 {
		t.Errorf("updated members = %d, want 3", len(memberRepo.updated))
	}
}

func TestAssignPayoutOrder_Manual_SkipsUnknownAndInactiveMembers(t *testing.T) {
	f := activeCirculoFund(3)
	active := activeMember(f.ID)
	inactive := &fund.FundMember{
		ID:     uuid.New(),
		FundID: f.ID,
		UserID: uuid.New(),
		Status: fund.MemberStatusSuspended,
	}
	memberRepo := &mockMemberRepo{members: []*fund.FundMember{active, inactive}}
	uc := circulo.NewAssignPayoutOrderUseCase(&mockCirculoRepo{}, memberRepo)

	err := uc.Execute(context.Background(), f, []circulo.MemberOrder{
		{MemberID: active.ID.String(), Order: 1},
		{MemberID: inactive.ID.String(), Order: 2},
		{MemberID: uuid.New().String(), Order: 3},
	}, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	assertPayoutOrder(t, active, 1)
	if inactive.PayoutOrder != nil {
		t.Errorf("inactive member should not receive payout order, got %d", *inactive.PayoutOrder)
	}
	if len(memberRepo.updated) != 1 {
		t.Errorf("updated members = %d, want 1", len(memberRepo.updated))
	}
}

func TestAssignPayoutOrder_Randomize_AssignsUniqueSequentialOrdersToActiveMembers(t *testing.T) {
	f := activeCirculoFund(4)
	activeA := activeMember(f.ID)
	activeB := activeMember(f.ID)
	activeC := activeMember(f.ID)
	inactive := &fund.FundMember{ID: uuid.New(), FundID: f.ID, UserID: uuid.New(), Status: fund.MemberStatusLeft}
	memberRepo := &mockMemberRepo{members: []*fund.FundMember{activeA, inactive, activeB, activeC}}
	uc := circulo.NewAssignPayoutOrderUseCase(&mockCirculoRepo{}, memberRepo)

	err := uc.Execute(context.Background(), f, nil, true)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(memberRepo.updated) != 3 {
		t.Fatalf("updated members = %d, want 3 active members", len(memberRepo.updated))
	}
	seen := map[int]bool{}
	for _, member := range []*fund.FundMember{activeA, activeB, activeC} {
		if member.PayoutOrder == nil {
			t.Fatalf("active member %s has nil PayoutOrder", member.ID)
		}
		order := *member.PayoutOrder
		if order < 1 || order > 3 {
			t.Fatalf("order = %d, want in [1,3]", order)
		}
		if seen[order] {
			t.Fatalf("duplicate payout order: %d", order)
		}
		seen[order] = true
	}
	if inactive.PayoutOrder != nil {
		t.Errorf("inactive member should not receive payout order, got %d", *inactive.PayoutOrder)
	}
}

func TestAssignPayoutOrder_WrongFundType_ReturnsError(t *testing.T) {
	f := activeCirculoFund(3)
	f.Type = fund.FundTypeVaca
	memberRepo := &mockMemberRepo{members: []*fund.FundMember{activeMember(f.ID)}}
	uc := circulo.NewAssignPayoutOrderUseCase(&mockCirculoRepo{}, memberRepo)

	err := uc.Execute(context.Background(), f, nil, true)
	assertAppErrorCode(t, err, "WRONG_FUND_TYPE")
	if len(memberRepo.updated) != 0 {
		t.Error("members should not be updated for wrong fund type")
	}
}

func TestAssignPayoutOrder_NoActiveMembers_ReturnsError(t *testing.T) {
	f := activeCirculoFund(3)
	memberRepo := &mockMemberRepo{members: []*fund.FundMember{
		{ID: uuid.New(), FundID: f.ID, Status: fund.MemberStatusLeft},
	}}
	uc := circulo.NewAssignPayoutOrderUseCase(&mockCirculoRepo{}, memberRepo)

	err := uc.Execute(context.Background(), f, nil, true)
	assertAppErrorCode(t, err, "NO_ACTIVE_MEMBERS")
}

func TestAssignPayoutOrder_InvalidMemberID_ReturnsError(t *testing.T) {
	f := activeCirculoFund(3)
	memberRepo := &mockMemberRepo{members: []*fund.FundMember{activeMember(f.ID)}}
	uc := circulo.NewAssignPayoutOrderUseCase(&mockCirculoRepo{}, memberRepo)

	err := uc.Execute(context.Background(), f, []circulo.MemberOrder{{MemberID: "not-a-uuid", Order: 1}}, false)
	assertAppErrorCode(t, err, "INVALID_MEMBER_ID")
	if len(memberRepo.updated) != 0 {
		t.Error("members should not be updated after invalid member_id")
	}
}

func TestAssignPayoutOrder_ListMembersError_ReturnsError(t *testing.T) {
	f := activeCirculoFund(3)
	wantErr := errors.New("list failed")
	uc := circulo.NewAssignPayoutOrderUseCase(&mockCirculoRepo{}, &mockMemberRepo{listErr: wantErr})

	err := uc.Execute(context.Background(), f, nil, true)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestAssignPayoutOrder_UpdateError_ReturnsError(t *testing.T) {
	f := activeCirculoFund(3)
	wantErr := errors.New("update failed")
	memberRepo := &mockMemberRepo{members: []*fund.FundMember{activeMember(f.ID)}, updateErr: wantErr}
	uc := circulo.NewAssignPayoutOrderUseCase(&mockCirculoRepo{}, memberRepo)

	err := uc.Execute(context.Background(), f, nil, true)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func assertPayoutOrder(t *testing.T, member *fund.FundMember, want int) {
	t.Helper()

	if member.PayoutOrder == nil {
		t.Fatalf("member %s PayoutOrder is nil, want %d", member.ID, want)
	}
	if *member.PayoutOrder != want {
		t.Fatalf("member %s PayoutOrder = %d, want %d", member.ID, *member.PayoutOrder, want)
	}
}

func assertAppErrorCode(t *testing.T, err error, wantCode string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error code %s, got nil", wantCode)
	}
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("error = %v, want AppError code %s", err, wantCode)
	}
	if appErr.Code != wantCode {
		t.Fatalf("error code = %s, want %s", appErr.Code, wantCode)
	}
}
