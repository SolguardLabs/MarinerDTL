package risk

import (
	"fmt"
	"sort"

	"marinerdtl/src/money"
)

type Decision string

const (
	DecisionApprove Decision = "approve"
	DecisionReview  Decision = "review"
	DecisionHold    Decision = "hold"
)

type LimitPolicy struct {
	MaximumCapitalShortfall money.Amount `json:"maximumCapitalShortfall"`
	MaximumRouteShareBps    int64        `json:"maximumRouteShareBps"`
	MaximumCustodianBps     int64        `json:"maximumCustodianBps"`
	MaximumOperatorBps      int64        `json:"maximumOperatorBps"`
	MaximumCorridorBps      int64        `json:"maximumCorridorBps"`
	MinimumCoverageBps      int64        `json:"minimumCoverageBps"`
}

func DefaultLimitPolicy() LimitPolicy {
	return LimitPolicy{
		MaximumCapitalShortfall: 0,
		MaximumRouteShareBps:    3500,
		MaximumCustodianBps:     4500,
		MaximumOperatorBps:      5000,
		MaximumCorridorBps:      6000,
		MinimumCoverageBps:      12000,
	}
}

func (p LimitPolicy) Validate() error {
	if err := p.MaximumCapitalShortfall.Validate(); err != nil {
		return fmt.Errorf("maximum capital shortfall: %w", err)
	}
	values := map[string]int64{
		"route": p.MaximumRouteShareBps, "custodian": p.MaximumCustodianBps,
		"operator": p.MaximumOperatorBps, "corridor": p.MaximumCorridorBps,
	}
	for name, value := range values {
		if value <= 0 || value > money.BasisMax {
			return fmt.Errorf("maximum %s share bps out of range", name)
		}
	}
	if p.MinimumCoverageBps < money.BasisMax || p.MinimumCoverageBps > 30000 {
		return fmt.Errorf("minimum coverage bps out of range")
	}
	return nil
}

type LimitBreach struct {
	Code     string       `json:"code"`
	Subject  string       `json:"subject"`
	Observed money.Amount `json:"observed"`
	Limit    money.Amount `json:"limit"`
	Unit     string       `json:"unit"`
	Severity Severity     `json:"severity"`
}

type GateResult struct {
	Decision Decision      `json:"decision"`
	Breaches []LimitBreach `json:"breaches"`
}

func EvaluateLimits(report Report, policy LimitPolicy) (GateResult, error) {
	if err := policy.Validate(); err != nil {
		return GateResult{}, err
	}
	result := GateResult{Decision: DecisionApprove, Breaches: []LimitBreach{}}
	if report.Capital.CapitalShortfall > policy.MaximumCapitalShortfall {
		result.Breaches = append(result.Breaches, LimitBreach{
			Code: "capital_shortfall", Subject: report.Scenario,
			Observed: report.Capital.CapitalShortfall, Limit: policy.MaximumCapitalShortfall,
			Unit: "amount", Severity: SeverityCritical,
		})
	}
	if report.Capital.CoverageBps < policy.MinimumCoverageBps {
		result.Breaches = append(result.Breaches, LimitBreach{
			Code: "coverage_below_limit", Subject: report.Scenario,
			Observed: money.Amount(report.Capital.CoverageBps), Limit: money.Amount(policy.MinimumCoverageBps),
			Unit: "bps", Severity: SeverityHigh,
		})
	}
	for _, route := range report.Routes {
		appendShareBreach(&result, "route_concentration", route.RouteID, route.PortfolioShareBps, policy.MaximumRouteShareBps)
	}
	for _, item := range report.Custodians {
		appendShareBreach(&result, "custodian_concentration", item.ID, item.ShareBps, policy.MaximumCustodianBps)
	}
	for _, item := range report.Operators {
		appendShareBreach(&result, "operator_concentration", item.ID, item.ShareBps, policy.MaximumOperatorBps)
	}
	for _, item := range report.Corridors {
		appendShareBreach(&result, "corridor_concentration", item.ID, item.ShareBps, policy.MaximumCorridorBps)
	}

	sort.Slice(result.Breaches, func(i, j int) bool {
		if result.Breaches[i].Code == result.Breaches[j].Code {
			return result.Breaches[i].Subject < result.Breaches[j].Subject
		}
		return result.Breaches[i].Code < result.Breaches[j].Code
	})
	for _, breach := range result.Breaches {
		if breach.Severity == SeverityCritical || breach.Severity == SeverityHigh {
			result.Decision = DecisionHold
			return result, nil
		}
		result.Decision = DecisionReview
	}
	return result, nil
}

func appendShareBreach(result *GateResult, code, subject string, observed, limit int64) {
	if observed <= limit {
		return
	}
	severity := SeverityModerate
	if observed > limit+1500 {
		severity = SeverityHigh
	}
	result.Breaches = append(result.Breaches, LimitBreach{
		Code: code, Subject: subject, Observed: money.Amount(observed), Limit: money.Amount(limit),
		Unit: "bps", Severity: severity,
	})
}

func (a *Analyzer) findings(report Report) []Finding {
	findings := []Finding{}
	if report.Capital.CapitalShortfall > 0 {
		findings = append(findings, Finding{
			Code: "capital_shortfall", Severity: SeverityCritical, Subject: report.Scenario,
			Message: fmt.Sprintf("capital shortfall is %s", report.Capital.CapitalShortfall),
		})
	}
	for _, route := range report.Routes {
		if route.PortfolioShareBps > a.scenario.MaximumSingleRouteShareBps {
			findings = append(findings, Finding{
				Code: "single_route_concentration", Severity: SeverityHigh, Subject: route.RouteID,
				Message: fmt.Sprintf("route represents %d bps of gross portfolio value", route.PortfolioShareBps),
			})
		}
	}
	for _, custodian := range report.Custodians {
		if custodian.ShareBps > 5000 {
			findings = append(findings, Finding{
				Code: "custodian_dependency", Severity: SeverityModerate, Subject: custodian.ID,
				Message: fmt.Sprintf("custodian represents %d bps of at-risk value", custodian.ShareBps),
			})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code == findings[j].Code {
			return findings[i].Subject < findings[j].Subject
		}
		return findings[i].Code < findings[j].Code
	})
	return findings
}
