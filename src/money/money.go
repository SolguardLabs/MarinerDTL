package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Amount int64

const (
	Zero     Amount = 0
	BasisMax int64  = 10000
)

var (
	ErrNegativeAmount = errors.New("amount cannot be negative")
	ErrOverflow       = errors.New("amount overflow")
	ErrDivisionByZero = errors.New("division by zero")
)

func New(value int64) (Amount, error) {
	if value < 0 {
		return 0, ErrNegativeAmount
	}
	return Amount(value), nil
}

func Must(value int64) Amount {
	amount, err := New(value)
	if err != nil {
		panic(err)
	}
	return amount
}

func Parse(raw string) (Amount, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("empty amount")
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return New(parsed)
}

func (a Amount) Int64() int64 {
	return int64(a)
}

func (a Amount) String() string {
	return strconv.FormatInt(int64(a), 10)
}

func (a Amount) IsZero() bool {
	return a == 0
}

func (a Amount) IsPositive() bool {
	return a > 0
}

func (a Amount) Validate() error {
	if a < 0 {
		return ErrNegativeAmount
	}
	return nil
}

func (a Amount) Cmp(other Amount) int {
	if a < other {
		return -1
	}
	if a > other {
		return 1
	}
	return 0
}

func (a Amount) Add(other Amount) (Amount, error) {
	if err := a.Validate(); err != nil {
		return 0, err
	}
	if err := other.Validate(); err != nil {
		return 0, err
	}
	if int64(a) > math.MaxInt64-int64(other) {
		return 0, ErrOverflow
	}
	return a + other, nil
}

func (a Amount) Sub(other Amount) (Amount, error) {
	if err := a.Validate(); err != nil {
		return 0, err
	}
	if err := other.Validate(); err != nil {
		return 0, err
	}
	if other > a {
		return 0, ErrNegativeAmount
	}
	return a - other, nil
}

func (a Amount) Mul(factor int64) (Amount, error) {
	if err := a.Validate(); err != nil {
		return 0, err
	}
	if factor < 0 {
		return 0, ErrNegativeAmount
	}
	if a == 0 || factor == 0 {
		return 0, nil
	}
	if int64(a) > math.MaxInt64/factor {
		return 0, ErrOverflow
	}
	return Amount(int64(a) * factor), nil
}

func (a Amount) Div(divisor int64) (Amount, error) {
	if err := a.Validate(); err != nil {
		return 0, err
	}
	if divisor == 0 {
		return 0, ErrDivisionByZero
	}
	if divisor < 0 {
		return 0, ErrNegativeAmount
	}
	return Amount(int64(a) / divisor), nil
}

func (a Amount) MulBps(bps int64) (Amount, error) {
	if err := a.Validate(); err != nil {
		return 0, err
	}
	if bps < 0 {
		return 0, ErrNegativeAmount
	}
	product, err := a.Mul(bps)
	if err != nil {
		return 0, err
	}
	return product.Div(BasisMax)
}

func (a Amount) CeilBps(bps int64) (Amount, error) {
	if err := a.Validate(); err != nil {
		return 0, err
	}
	if bps < 0 {
		return 0, ErrNegativeAmount
	}
	if a == 0 || bps == 0 {
		return 0, nil
	}
	if int64(a) > math.MaxInt64/bps {
		return 0, ErrOverflow
	}
	product := int64(a) * bps
	return Amount((product + BasisMax - 1) / BasisMax), nil
}

func (a Amount) MustAdd(other Amount) Amount {
	result, err := a.Add(other)
	if err != nil {
		panic(err)
	}
	return result
}

func (a Amount) MustSub(other Amount) Amount {
	result, err := a.Sub(other)
	if err != nil {
		panic(err)
	}
	return result
}

func (a Amount) MustMulBps(bps int64) Amount {
	result, err := a.MulBps(bps)
	if err != nil {
		panic(err)
	}
	return result
}

func Max(values ...Amount) Amount {
	var out Amount
	for _, value := range values {
		if value > out {
			out = value
		}
	}
	return out
}

func Min(values ...Amount) Amount {
	if len(values) == 0 {
		return 0
	}
	out := values[0]
	for _, value := range values[1:] {
		if value < out {
			out = value
		}
	}
	return out
}

func Sum(values ...Amount) (Amount, error) {
	var total Amount
	for _, value := range values {
		next, err := total.Add(value)
		if err != nil {
			return 0, err
		}
		total = next
	}
	return total, nil
}

func Clamp(value, min, max Amount) Amount {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func RatioBps(numerator, denominator Amount) int64 {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	if numerator >= denominator {
		return BasisMax
	}
	return int64(numerator) * BasisMax / int64(denominator)
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a *Amount) UnmarshalJSON(data []byte) error {
	text := strings.TrimSpace(string(data))
	if text == "null" || text == "" {
		*a = 0
		return nil
	}
	if strings.HasPrefix(text, "\"") {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		parsed, err := Parse(raw)
		if err != nil {
			return err
		}
		*a = parsed
		return nil
	}
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return err
	}
	amount, err := New(parsed)
	if err != nil {
		return err
	}
	*a = amount
	return nil
}

func MapCopy(in map[string]Amount) map[string]Amount {
	out := make(map[string]Amount, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func AddToMap(values map[string]Amount, key string, amount Amount) error {
	if values == nil {
		return fmt.Errorf("nil amount map")
	}
	current := values[key]
	next, err := current.Add(amount)
	if err != nil {
		return err
	}
	values[key] = next
	return nil
}

func SubFromMap(values map[string]Amount, key string, amount Amount) error {
	if values == nil {
		return fmt.Errorf("nil amount map")
	}
	current := values[key]
	next, err := current.Sub(amount)
	if err != nil {
		return err
	}
	values[key] = next
	return nil
}
