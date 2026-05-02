package vaca_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/usecase/vaca"
	"github.com/shopspring/decimal"
)

type mockVacaRepo struct{ cfg *fund.VacaConfig }

func (m *mockVacaRepo) Create(_ context.Context, v *fund.VacaConfig) error { return nil }
func (m *mockVacaRepo) GetByFundID(_ context.Context, _ uuid.UUID) (*fund.VacaConfig, error) {
	return m.cfg, nil
}
func (m *mockVacaRepo) Update(_ context.Context, _ *fund.VacaConfig) error { return nil }

type mockBalanceReader struct{ balance decimal.Decimal }

func (m *mockBalanceReader) GetFundBalance(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
	return m.balance, nil
}

func vacaFund() *fund.Fund {
	return &fund.Fund{ID: uuid.New(), Type: fund.FundTypeVaca, Status: fund.FundStatusActive}
}

func TestGetProgress_Percentage(t *testing.T) {
	tests := []struct {
		name        string
		goal        string
		balance     string
		wantPct     string
		wantReached bool
	}{
		{"50% progress", "1000000", "500000", "50.00", false},
		{"100% — goal reached", "1000000", "1000000", "100.00", true},
		{"over 100%", "1000000", "1200000", "120.00", true},
		{"zero balance", "1000000", "0", "0.00", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			goal, _    := decimal.NewFromString(tc.goal)
			balance, _ := decimal.NewFromString(tc.balance)

			uc := vaca.NewGetProgressUseCase(
				&mockVacaRepo{cfg: &fund.VacaConfig{
					FundID:           uuid.New(),
					GoalAmount:       goal,
					GoalDescription:  "test goal",
					DistributionType: "goal_reached",
				}},
				&mockBalanceReader{balance: balance},
			)

			result, err := uc.Execute(context.Background(), vacaFund())
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if result.Percentage.StringFixed(2) != tc.wantPct {
				t.Errorf("Percentage = %s, want %s", result.Percentage.StringFixed(2), tc.wantPct)
			}
			if result.GoalReached != tc.wantReached {
				t.Errorf("GoalReached = %v, want %v", result.GoalReached, tc.wantReached)
			}
		})
	}
}

func TestGetProgress_WrongFundType(t *testing.T) {
	uc := vaca.NewGetProgressUseCase(&mockVacaRepo{}, &mockBalanceReader{})
	f := &fund.Fund{ID: uuid.New(), Type: fund.FundTypeCirculo}
	_, err := uc.Execute(context.Background(), f)
	if err == nil {
		t.Error("expected error for non-vaca fund type")
	}
}
