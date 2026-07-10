package settlement

import (
	"fmt"

	"marinerdtl/src/domain"
	"marinerdtl/src/money"
	"marinerdtl/src/policy"
)

type DisputePlan struct {
	RouteID      string
	MilestoneID  string
	ClaimantID   string
	FreezeAmount money.Amount
	PenaltyBps   int64
	Reason       string
}

func BuildDisputePlan(route *domain.Route, milestone *domain.Milestone, claimant *domain.Account, requested money.Amount, reason string, rules policy.Policy) (DisputePlan, error) {
	if route == nil || milestone == nil || claimant == nil {
		return DisputePlan{}, fmt.Errorf("route, milestone and claimant are required")
	}
	if route.IsTerminal() {
		return DisputePlan{}, fmt.Errorf("route %s is terminal", route.ID)
	}
	if milestone.RouteID != route.ID {
		return DisputePlan{}, fmt.Errorf("milestone route mismatch")
	}
	if milestone.IsTerminal() {
		return DisputePlan{}, fmt.Errorf("milestone %s is terminal", milestone.ID)
	}
	if milestone.DisputeID != "" {
		return DisputePlan{}, fmt.Errorf("milestone %s already has an active dispute", milestone.ID)
	}
	freeze, err := rules.DisputeFreezeAmount(milestone, requested)
	if err != nil {
		return DisputePlan{}, err
	}
	penaltyBps, err := rules.NormalizePenaltyBps(milestone.PenaltyBps)
	if err != nil {
		return DisputePlan{}, err
	}
	return DisputePlan{
		RouteID:      route.ID,
		MilestoneID:  milestone.ID,
		ClaimantID:   claimant.ID,
		FreezeAmount: freeze,
		PenaltyBps:   penaltyBps,
		Reason:       reason,
	}, nil
}

type CancellationPlan struct {
	RouteID      string
	MilestoneID  string
	Refund       money.Amount
	Penalty      money.Amount
	PenaltyBps   int64
	Beneficiary  string
	FullRouteEnd bool
}

func BuildCancellationPlan(route *domain.Route, milestone *domain.Milestone, rules policy.Policy, penaltyBps int64) (CancellationPlan, error) {
	if route == nil || milestone == nil {
		return CancellationPlan{}, fmt.Errorf("route and milestone are required")
	}
	if !rules.AllowRouteCancellation {
		return CancellationPlan{}, fmt.Errorf("route cancellation disabled")
	}
	if route.Status == domain.RouteCompleted || route.Status == domain.RouteCancelled {
		return CancellationPlan{}, fmt.Errorf("route %s is terminal", route.ID)
	}
	if milestone.RouteID != route.ID {
		return CancellationPlan{}, fmt.Errorf("milestone route mismatch")
	}
	if milestone.Status == domain.MilestoneReleased || milestone.Status == domain.MilestoneCancelled {
		return CancellationPlan{}, fmt.Errorf("milestone %s is terminal", milestone.ID)
	}
	if penaltyBps == 0 {
		penaltyBps = milestone.PenaltyBps
	}
	refund, penalty, err := rules.CancellationSplit(milestone, penaltyBps)
	if err != nil {
		return CancellationPlan{}, err
	}
	return CancellationPlan{
		RouteID:     route.ID,
		MilestoneID: milestone.ID,
		Refund:      refund,
		Penalty:     penalty,
		PenaltyBps:  penaltyBps,
		Beneficiary: milestone.CarrierID,
	}, nil
}

func ResolutionStatus(raw string) (domain.DisputeStatus, error) {
	switch raw {
	case "", "resolved":
		return domain.DisputeResolved, nil
	case "accepted":
		return domain.DisputeAccepted, nil
	case "rejected":
		return domain.DisputeRejected, nil
	case "cancelled":
		return domain.DisputeCancelled, nil
	default:
		return "", fmt.Errorf("unknown dispute resolution status %q", raw)
	}
}
