package circulo

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/pkg/apperror"
)

// MemberOrder pairs a member ID with their desired payout position.
type MemberOrder struct {
	MemberID string `json:"member_id"`
	Order    int    `json:"order"`
}

type AssignPayoutOrderUseCase struct {
	circulo fund.CirculoRepository
	members fund.MemberRepository
}

func NewAssignPayoutOrderUseCase(circulo fund.CirculoRepository, members fund.MemberRepository) *AssignPayoutOrderUseCase {
	return &AssignPayoutOrderUseCase{circulo: circulo, members: members}
}

// Execute assigns payout_order to each active member of a circulo fund.
// If randomize=true the order is shuffled randomly; otherwise assignments are applied manually.
func (uc *AssignPayoutOrderUseCase) Execute(
	ctx context.Context,
	f *fund.Fund,
	assignments []MemberOrder,
	randomize bool,
) error {
	if f.Type != fund.FundTypeCirculo {
		return apperror.New(400, "WRONG_FUND_TYPE", "this fund is not a circulo")
	}

	members, err := uc.members.ListByFund(ctx, f.ID)
	if err != nil {
		return err
	}

	active := make([]*fund.FundMember, 0, len(members))
	for _, m := range members {
		if m.Status == fund.MemberStatusActive {
			active = append(active, m)
		}
	}
	if len(active) == 0 {
		return apperror.New(400, "NO_ACTIVE_MEMBERS", "fund has no active members")
	}

	if randomize {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(active), func(i, j int) { active[i], active[j] = active[j], active[i] })
		for i, m := range active {
			order := i + 1
			m.PayoutOrder = &order
			if err := uc.members.Update(ctx, m); err != nil {
				return err
			}
		}
		return nil
	}

	// Manual: build lookup map from active members only and apply each assignment.
	memberByID := make(map[uuid.UUID]*fund.FundMember, len(active))
	for _, m := range active {
		memberByID[m.ID] = m
	}
	for _, a := range assignments {
		id, err := uuid.Parse(a.MemberID)
		if err != nil {
			return apperror.New(400, "INVALID_MEMBER_ID", "invalid member_id: "+a.MemberID)
		}
		m, ok := memberByID[id]
		if !ok {
			continue // silently skip unknown member IDs
		}
		order := a.Order
		m.PayoutOrder = &order
		if err := uc.members.Update(ctx, m); err != nil {
			return err
		}
	}
	return nil
}
