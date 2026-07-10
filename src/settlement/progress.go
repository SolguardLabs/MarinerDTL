package settlement

import (
	"fmt"

	"marinerdtl/src/domain"
	"marinerdtl/src/money"
)

type Progress struct {
	RouteID        string
	Total          money.Amount
	Released       money.Amount
	Cancelled      money.Amount
	Penalized      money.Amount
	Frozen         money.Amount
	Certified      money.Amount
	Open           money.Amount
	DeliveredBps   int64
	CancelledBps   int64
	CertifiedBps   int64
	ProtectedBps   int64
	TerminalCount  int
	MilestoneCount int
}

func CalculateProgress(route *domain.Route, milestones []*domain.Milestone) Progress {
	progress := Progress{RouteID: route.ID, MilestoneCount: len(milestones)}
	for _, milestone := range milestones {
		progress.Total = progress.Total.MustAdd(milestone.Amount)
		progress.Released = progress.Released.MustAdd(milestone.ReleasedAmount)
		progress.Cancelled = progress.Cancelled.MustAdd(milestone.CancelledAmount)
		progress.Penalized = progress.Penalized.MustAdd(milestone.PenaltyAmount)
		progress.Frozen = progress.Frozen.MustAdd(milestone.FrozenAmount)
		if milestone.CertificateID != "" {
			progress.Certified = progress.Certified.MustAdd(milestone.RemainingAmount()).MustAdd(milestone.ReleasedAmount)
		}
		if milestone.IsTerminal() {
			progress.TerminalCount++
		}
		progress.Open = progress.Open.MustAdd(milestone.RemainingAmount())
	}
	progress.DeliveredBps = money.RatioBps(progress.Released, progress.Total)
	progress.CancelledBps = money.RatioBps(progress.Cancelled, progress.Total)
	progress.CertifiedBps = money.RatioBps(progress.Certified, progress.Total)
	progress.ProtectedBps = money.RatioBps(progress.Frozen, progress.Total)
	return progress
}

func (p Progress) IsFullyTerminal() bool {
	return p.MilestoneCount > 0 && p.TerminalCount == p.MilestoneCount
}

func (p Progress) HasEconomicActivity() bool {
	return p.Released > 0 || p.Cancelled > 0 || p.Penalized > 0
}

func (p Progress) ValidateAgainstRoute(route *domain.Route) error {
	if p.Total == 0 {
		return fmt.Errorf("route %s has no milestone value", route.ID)
	}
	if route.EscrowTotal < p.Total {
		return fmt.Errorf("route %s escrow is below milestone value", route.ID)
	}
	if p.Released.MustAdd(p.Cancelled).MustAdd(p.Penalized).MustAdd(p.Open) < p.Total {
		return fmt.Errorf("route %s milestone accounting is incomplete", route.ID)
	}
	return nil
}

type ReleasePlan struct {
	RouteID     string
	MilestoneID string
	Asset       string
	Amount      money.Amount
	CarrierID   string
	CustodianID string
}

func BuildReleasePlan(route *domain.Route, milestone *domain.Milestone, amount money.Amount) (ReleasePlan, error) {
	if route == nil || milestone == nil {
		return ReleasePlan{}, fmt.Errorf("route and milestone are required")
	}
	if milestone.RouteID != route.ID {
		return ReleasePlan{}, fmt.Errorf("milestone route mismatch")
	}
	if !milestone.CanRelease() {
		return ReleasePlan{}, fmt.Errorf("milestone %s cannot be released in status %s", milestone.ID, milestone.Status)
	}
	if amount == 0 {
		amount = milestone.RemainingAmount()
	}
	if amount <= 0 || amount > milestone.RemainingAmount() {
		return ReleasePlan{}, fmt.Errorf("invalid release amount")
	}
	return ReleasePlan{
		RouteID:     route.ID,
		MilestoneID: milestone.ID,
		Asset:       route.Asset,
		Amount:      amount,
		CarrierID:   milestone.CarrierID,
		CustodianID: milestone.CustodianID,
	}, nil
}
