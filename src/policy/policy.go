package policy

import (
	"fmt"
	"strings"

	"marinerdtl/src/domain"
	"marinerdtl/src/money"
)

type Policy struct {
	Name                       string       `json:"name"`
	MinimumDeposit             money.Amount `json:"minimumDeposit"`
	MinimumEscrowFunding       money.Amount `json:"minimumEscrowFunding"`
	MinimumMilestoneAmount     money.Amount `json:"minimumMilestoneAmount"`
	MaximumRouteEscrow         money.Amount `json:"maximumRouteEscrow"`
	MaximumMilestonesPerRoute  int          `json:"maximumMilestonesPerRoute"`
	MaximumRebateBudget        money.Amount `json:"maximumRebateBudget"`
	MaximumRebateBps           int64        `json:"maximumRebateBps"`
	DefaultMinProgressBps      int64        `json:"defaultMinProgressBps"`
	DefaultPenaltyBps          int64        `json:"defaultPenaltyBps"`
	MaximumPenaltyBps          int64        `json:"maximumPenaltyBps"`
	DisputeFreezeBps           int64        `json:"disputeFreezeBps"`
	MinimumCertificateHashSize int          `json:"minimumCertificateHashSize"`
	RequireCustodianRole       bool         `json:"requireCustodianRole"`
	AllowPartialDisputes       bool         `json:"allowPartialDisputes"`
	AllowRouteCancellation     bool         `json:"allowRouteCancellation"`
}

func Default() Policy {
	return Policy{
		Name:                       "standard-maritime-settlement",
		MinimumDeposit:             money.Must(1),
		MinimumEscrowFunding:       money.Must(1),
		MinimumMilestoneAmount:     money.Must(1),
		MaximumRouteEscrow:         money.Must(100_000_000_000),
		MaximumMilestonesPerRoute:  24,
		MaximumRebateBudget:        money.Must(10_000_000_000),
		MaximumRebateBps:           2500,
		DefaultMinProgressBps:      3000,
		DefaultPenaltyBps:          800,
		MaximumPenaltyBps:          5000,
		DisputeFreezeBps:           10000,
		MinimumCertificateHashSize: 12,
		RequireCustodianRole:       true,
		AllowPartialDisputes:       true,
		AllowRouteCancellation:     true,
	}
}

func (p Policy) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("policy name is required")
	}
	if p.MinimumDeposit < 0 || p.MinimumEscrowFunding < 0 || p.MinimumMilestoneAmount < 0 {
		return fmt.Errorf("minimum amounts cannot be negative")
	}
	if p.MaximumRouteEscrow <= 0 {
		return fmt.Errorf("maximum route escrow must be positive")
	}
	if p.MaximumMilestonesPerRoute <= 0 {
		return fmt.Errorf("maximum milestones per route must be positive")
	}
	if p.MaximumRebateBudget < 0 {
		return fmt.Errorf("maximum rebate budget cannot be negative")
	}
	if p.MaximumRebateBps < 0 || p.MaximumRebateBps > money.BasisMax {
		return fmt.Errorf("maximum rebate bps out of range")
	}
	if p.DefaultMinProgressBps < 0 || p.DefaultMinProgressBps > money.BasisMax {
		return fmt.Errorf("default progress bps out of range")
	}
	if p.DefaultPenaltyBps < 0 || p.DefaultPenaltyBps > p.MaximumPenaltyBps {
		return fmt.Errorf("default penalty bps out of range")
	}
	if p.MaximumPenaltyBps < 0 || p.MaximumPenaltyBps > money.BasisMax {
		return fmt.Errorf("maximum penalty bps out of range")
	}
	if p.DisputeFreezeBps <= 0 || p.DisputeFreezeBps > money.BasisMax {
		return fmt.Errorf("dispute freeze bps out of range")
	}
	if p.MinimumCertificateHashSize < 8 {
		return fmt.Errorf("certificate hash size too small")
	}
	return nil
}

func (p Policy) CheckDeposit(amount money.Amount) error {
	if amount < p.MinimumDeposit {
		return fmt.Errorf("deposit below minimum")
	}
	return amount.Validate()
}

func (p Policy) CheckFunding(route *domain.Route, amount money.Amount) error {
	if amount < p.MinimumEscrowFunding {
		return fmt.Errorf("escrow funding below minimum")
	}
	next, err := route.EscrowTotal.Add(amount)
	if err != nil {
		return err
	}
	if next > p.MaximumRouteEscrow {
		return fmt.Errorf("route escrow limit exceeded")
	}
	return nil
}

func (p Policy) CheckMilestone(route *domain.Route, amount money.Amount) error {
	if len(route.Milestones) >= p.MaximumMilestonesPerRoute {
		return fmt.Errorf("route milestone limit exceeded")
	}
	if amount < p.MinimumMilestoneAmount {
		return fmt.Errorf("milestone amount below minimum")
	}
	return amount.Validate()
}

func (p Policy) NormalizePenaltyBps(raw int64) (int64, error) {
	if raw == 0 {
		raw = p.DefaultPenaltyBps
	}
	if raw < 0 || raw > p.MaximumPenaltyBps {
		return 0, fmt.Errorf("penalty bps out of range")
	}
	return raw, nil
}

func (p Policy) NormalizeProgressBps(raw int64) (int64, error) {
	if raw == 0 {
		raw = p.DefaultMinProgressBps
	}
	if raw < 0 || raw > money.BasisMax {
		return 0, fmt.Errorf("progress bps out of range")
	}
	return raw, nil
}

func (p Policy) CheckRebateBudget(amount money.Amount) error {
	if amount < 0 {
		return fmt.Errorf("rebate budget cannot be negative")
	}
	if amount > p.MaximumRebateBudget {
		return fmt.Errorf("rebate budget exceeds policy maximum")
	}
	return nil
}

func (p Policy) CheckRebateBps(bps int64) error {
	if bps < 0 || bps > p.MaximumRebateBps {
		return fmt.Errorf("rebate bps out of range")
	}
	return nil
}

func (p Policy) CheckCertificateHash(raw string) error {
	value := strings.TrimSpace(raw)
	if len(value) < p.MinimumCertificateHashSize {
		return fmt.Errorf("certificate document hash too short")
	}
	for _, r := range value {
		if r == ' ' || r == '\t' || r == '\n' {
			return fmt.Errorf("certificate document hash contains whitespace")
		}
	}
	return nil
}

func (p Policy) DisputeFreezeAmount(milestone *domain.Milestone, requested money.Amount) (money.Amount, error) {
	remaining := milestone.RemainingAmount()
	if remaining <= 0 {
		return 0, fmt.Errorf("milestone has no disputable amount")
	}
	if requested == 0 {
		return remaining.MulBps(p.DisputeFreezeBps)
	}
	if requested > remaining {
		return 0, fmt.Errorf("dispute amount exceeds milestone remaining amount")
	}
	if !p.AllowPartialDisputes && requested != remaining {
		return 0, fmt.Errorf("partial disputes disabled")
	}
	return requested, nil
}

func (p Policy) CancellationSplit(milestone *domain.Milestone, penaltyBps int64) (refund money.Amount, penalty money.Amount, err error) {
	remaining := milestone.RemainingAmount()
	if remaining == 0 {
		return 0, 0, nil
	}
	if penaltyBps == 0 {
		penaltyBps = milestone.PenaltyBps
	}
	if penaltyBps < 0 || penaltyBps > p.MaximumPenaltyBps {
		return 0, 0, fmt.Errorf("cancellation penalty bps out of range")
	}
	penalty, err = remaining.MulBps(penaltyBps)
	if err != nil {
		return 0, 0, err
	}
	refund, err = remaining.Sub(penalty)
	if err != nil {
		return 0, 0, err
	}
	return refund, penalty, nil
}

func (p Policy) ShouldCompleteRoute(state interface {
	IsRouteComplete(string) bool
}, route *domain.Route) bool {
	if route == nil || route.IsTerminal() {
		return false
	}
	return state.IsRouteComplete(route.ID)
}
