package settlement

import (
	"fmt"

	"marinerdtl/src/domain"
	"marinerdtl/src/money"
	"marinerdtl/src/policy"
)

type RebatePlan struct {
	RouteID         string       `json:"routeId"`
	Asset           string       `json:"asset"`
	TreasuryID      string       `json:"treasuryId"`
	RecipientID     string       `json:"recipientId"`
	GrossBudget     money.Amount `json:"grossBudget"`
	AlreadyClaimed  money.Amount `json:"alreadyClaimed"`
	ClaimableAmount money.Amount `json:"claimableAmount"`
	ProgressBps     int64        `json:"progressBps"`
	RequiredBps     int64        `json:"requiredBps"`
}

func BuildRebatePlan(route *domain.Route, milestones []*domain.Milestone, treasuryID string, rules policy.Policy) (RebatePlan, error) {
	if route == nil {
		return RebatePlan{}, fmt.Errorf("route is required")
	}
	if route.Status == domain.RouteCancelled || route.Status == domain.RouteSuspended {
		return RebatePlan{}, fmt.Errorf("route %s is not eligible for rebates", route.ID)
	}
	if route.RebateBudget == 0 {
		return RebatePlan{}, fmt.Errorf("route %s has no rebate budget", route.ID)
	}
	if err := rules.CheckRebateBudget(route.RebateBudget); err != nil {
		return RebatePlan{}, err
	}
	progress := CalculateProgress(route, milestones)
	if progress.Total == 0 {
		return RebatePlan{}, fmt.Errorf("route %s has no milestone value", route.ID)
	}
	required := route.MinProgressBps
	if required == 0 {
		required = rules.DefaultMinProgressBps
	}
	if progress.DeliveredBps < required {
		return RebatePlan{}, fmt.Errorf("route %s progress below rebate threshold", route.ID)
	}
	outstanding := route.OutstandingRebate()
	if outstanding == 0 {
		return RebatePlan{}, fmt.Errorf("route %s rebate already claimed", route.ID)
	}
	return RebatePlan{
		RouteID:         route.ID,
		Asset:           route.Asset,
		TreasuryID:      treasuryID,
		RecipientID:     route.RebateAccountID,
		GrossBudget:     route.RebateBudget,
		AlreadyClaimed:  route.RebateClaimed,
		ClaimableAmount: outstanding,
		ProgressBps:     progress.DeliveredBps,
		RequiredBps:     required,
	}, nil
}
