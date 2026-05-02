package payment

import (
	"context"
	"time"

	"github.com/sacramento-finance/backend/internal/domain/fund"
	"github.com/sacramento-finance/backend/internal/domain/ledger"
	"github.com/sacramento-finance/backend/pkg/idgen"
)

// PaymentRepository is the subset of ledger.PaymentRepository needed here.
type PaymentRepository interface {
	CreateBatch(ctx context.Context, payments []*ledger.Payment) error
}

type GenerateScheduleUseCase struct {
	payments PaymentRepository
}

func NewGenerateScheduleUseCase(payments PaymentRepository) *GenerateScheduleUseCase {
	return &GenerateScheduleUseCase{payments: payments}
}

// Execute creates one Payment row per (member × period) when a fund is activated.
func (uc *GenerateScheduleUseCase) Execute(ctx context.Context, f *fund.Fund, members []*fund.FundMember) error {
	now := time.Now().UTC()
	var batch []*ledger.Payment

	for _, m := range members {
		if m.Status != fund.MemberStatusActive {
			continue
		}
		for period := 1; period <= f.Rules.TotalPeriods; period++ {
			due := dueDate(f.Rules.StartDate, f.Rules.Frequency, period)
			batch = append(batch, &ledger.Payment{
				ID:           idgen.New(),
				FundID:       f.ID,
				MemberID:     m.ID,
				PeriodNumber: period,
				DueDate:      due,
				AmountDue:    f.Rules.ContributionAmount,
				Status:       ledger.PaymentStatusPending,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}
	}

	if len(batch) == 0 {
		return nil
	}
	return uc.payments.CreateBatch(ctx, batch)
}

// dueDate computes when period N is due based on start date and frequency.
// Period 1 is due on start_date itself.
func dueDate(start time.Time, freq fund.PeriodFrequency, period int) time.Time {
	n := period - 1
	switch freq {
	case fund.FrequencyWeekly:
		return start.AddDate(0, 0, n*7)
	case fund.FrequencyBiweekly:
		return start.AddDate(0, 0, n*14)
	default: // monthly
		return start.AddDate(0, n, 0)
	}
}
