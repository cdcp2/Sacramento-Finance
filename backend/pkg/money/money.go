package money

import "github.com/shopspring/decimal"

// Zero is a convenience zero-value decimal.
var Zero = decimal.Zero

// New creates a Decimal from a float. Prefer FromString for user input.
func New(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f)
}

// FromString parses a string (safe for user input).
func FromString(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

// MustFromString panics on parse error — use only for hardcoded constants.
func MustFromString(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic("money.MustFromString: " + err.Error())
	}
	return d
}

// Percentage calculates pct% of amount (e.g. Percentage(1000, 10) = 100).
func Percentage(amount, pct decimal.Decimal) decimal.Decimal {
	return amount.Mul(pct).Div(decimal.NewFromInt(100))
}
