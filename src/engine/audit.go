package engine

import (
	"fmt"

	"marinerdtl/src/domain"
	"marinerdtl/src/settlement"
)

func AuditState(state *domain.State) []domain.AuditIssue {
	issues := []domain.AuditIssue{}
	issues = append(issues, auditRoutes(state)...)
	issues = append(issues, auditMilestones(state)...)
	issues = append(issues, auditAccounts(state)...)
	return issues
}

func auditRoutes(state *domain.State) []domain.AuditIssue {
	issues := []domain.AuditIssue{}
	for _, routeID := range state.SortedRouteIDs() {
		route := state.Routes[routeID]
		milestones := state.RouteMilestones(route.ID)
		progress := settlement.CalculateProgress(route, milestones)
		if len(milestones) == 0 {
			issues = append(issues, domain.AuditIssue{
				Code:     "route_without_milestones",
				Severity: "info",
				Message:  fmt.Sprintf("route %s has no milestones", route.ID),
			})
			continue
		}
		if progress.Total > route.EscrowTotal {
			issues = append(issues, domain.AuditIssue{
				Code:     "escrow_below_milestones",
				Severity: "high",
				Message:  fmt.Sprintf("route %s escrow total below milestone value", route.ID),
			})
		}
		if route.EscrowRemaining < 0 {
			issues = append(issues, domain.AuditIssue{
				Code:     "negative_escrow",
				Severity: "critical",
				Message:  fmt.Sprintf("route %s has negative escrow", route.ID),
			})
		}
		if route.RebateClaimed > route.RebateBudget {
			issues = append(issues, domain.AuditIssue{
				Code:     "rebate_budget_exceeded",
				Severity: "critical",
				Message:  fmt.Sprintf("route %s rebate budget exceeded", route.ID),
			})
		}
	}
	return issues
}

func auditMilestones(state *domain.State) []domain.AuditIssue {
	issues := []domain.AuditIssue{}
	for _, milestoneID := range state.SortedMilestoneIDs() {
		milestone := state.Milestones[milestoneID]
		if _, err := state.RequireRoute(milestone.RouteID); err != nil {
			issues = append(issues, domain.AuditIssue{
				Code:     "orphan_milestone",
				Severity: "high",
				Message:  fmt.Sprintf("milestone %s has missing route", milestone.ID),
			})
		}
		if milestone.ReleasedAmount.MustAdd(milestone.CancelledAmount).MustAdd(milestone.PenaltyAmount) > milestone.Amount {
			issues = append(issues, domain.AuditIssue{
				Code:     "milestone_overallocated",
				Severity: "critical",
				Message:  fmt.Sprintf("milestone %s exceeds nominal amount", milestone.ID),
			})
		}
		if milestone.Status == domain.MilestoneReleased && milestone.CertificateID == "" {
			issues = append(issues, domain.AuditIssue{
				Code:     "release_without_certificate",
				Severity: "critical",
				Message:  fmt.Sprintf("milestone %s released without certificate", milestone.ID),
			})
		}
		if milestone.Status == domain.MilestoneDisputed && milestone.FrozenAmount == 0 {
			issues = append(issues, domain.AuditIssue{
				Code:     "empty_dispute_freeze",
				Severity: "medium",
				Message:  fmt.Sprintf("milestone %s disputed without frozen amount", milestone.ID),
			})
		}
	}
	return issues
}

func auditAccounts(state *domain.State) []domain.AuditIssue {
	issues := []domain.AuditIssue{}
	for _, accountID := range state.SortedAccountIDs() {
		account := state.Accounts[accountID]
		for _, asset := range account.Assets() {
			if account.Available(asset) < 0 || account.ReservedBalance(asset) < 0 {
				issues = append(issues, domain.AuditIssue{
					Code:     "negative_account_balance",
					Severity: "critical",
					Message:  fmt.Sprintf("account %s has negative %s balance", account.ID, asset),
				})
			}
		}
	}
	return issues
}
