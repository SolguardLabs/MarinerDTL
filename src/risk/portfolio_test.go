package risk

import (
	"testing"

	"marinerdtl/src/domain"
	"marinerdtl/src/money"
)

func TestProjectStateBuildsDeterministicPositions(t *testing.T) {
	state := sampleState()
	positions, err := ProjectState(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(positions) != 2 || positions[0].RouteID != "route-a" || positions[1].RouteID != "route-b" {
		t.Fatalf("unexpected route order: %+v", positions)
	}
	if positions[0].Corridor != "ESBCN>NLRTM" {
		t.Fatalf("unexpected corridor: %s", positions[0].Corridor)
	}
	if positions[0].UnsettledValue != 800_000 {
		t.Fatalf("unexpected unsettled value: %s", positions[0].UnsettledValue)
	}
	if positions[0].CustodianValues["custody-a"] != 800_000 {
		t.Fatalf("unexpected custodian exposure: %+v", positions[0].CustodianValues)
	}
}

func TestOperatingScenarioComputesCapitalComponents(t *testing.T) {
	report, err := MustAnalyzer(OperatingScenario()).AnalyzeState(sampleState(), money.Must(350_000))
	if err != nil {
		t.Fatal(err)
	}
	if report.GrossValue != 2_000_000 {
		t.Fatalf("unexpected gross value: %s", report.GrossValue)
	}
	if report.Capital.RequiredCapital <= 0 || report.Capital.TargetCapital <= report.Capital.RequiredCapital {
		t.Fatalf("invalid capital calculation: %+v", report.Capital)
	}
	if len(report.Digest) != 64 {
		t.Fatalf("invalid digest: %q", report.Digest)
	}
}

func TestSevereScenarioChargesMoreThanOperating(t *testing.T) {
	state := sampleState()
	operating, err := MustAnalyzer(OperatingScenario()).AnalyzeState(state, money.Must(500_000))
	if err != nil {
		t.Fatal(err)
	}
	severe, err := MustAnalyzer(SevereScenario()).AnalyzeState(state, money.Must(500_000))
	if err != nil {
		t.Fatal(err)
	}
	if severe.Capital.RequiredCapital <= operating.Capital.RequiredCapital {
		t.Fatalf("severe capital %s must exceed operating capital %s", severe.Capital.RequiredCapital, operating.Capital.RequiredCapital)
	}
}

func TestDigestIsStableForSameInput(t *testing.T) {
	analyzer := MustAnalyzer(OperatingScenario())
	left, err := analyzer.AnalyzeState(sampleState(), money.Must(200_000))
	if err != nil {
		t.Fatal(err)
	}
	right, err := analyzer.AnalyzeState(sampleState(), money.Must(200_000))
	if err != nil {
		t.Fatal(err)
	}
	if left.Digest != right.Digest {
		t.Fatalf("digest changed: %s != %s", left.Digest, right.Digest)
	}
}

func TestConcentrationIsSortedByExposure(t *testing.T) {
	report, err := MustAnalyzer(OperatingScenario()).AnalyzeState(sampleState(), money.Must(400_000))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Operators) != 2 || report.Operators[0].Exposure < report.Operators[1].Exposure {
		t.Fatalf("operators not sorted: %+v", report.Operators)
	}
}

func TestLimitGateHoldsConcentratedPortfolio(t *testing.T) {
	report, err := MustAnalyzer(SevereScenario()).AnalyzeState(sampleState(), money.Must(1))
	if err != nil {
		t.Fatal(err)
	}
	gate, err := EvaluateLimits(report, DefaultLimitPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if gate.Decision != DecisionHold || len(gate.Breaches) == 0 {
		t.Fatalf("unexpected gate: %+v", gate)
	}
}

func TestBalancedStandalonePositionsCanPassLimits(t *testing.T) {
	positions := []RoutePosition{}
	for i := 0; i < 4; i++ {
		positions = append(positions, RoutePosition{
			RouteID: string(rune('a' + i)), Asset: "usdc", Corridor: string(rune('k' + i)),
			OperatorID: string(rune('o' + i)), GrossValue: 1_000_000,
			EscrowRemaining: 1_000_000, UnsettledValue: 1_000_000,
			CustodianValues: map[string]money.Amount{string(rune('c' + i)): 1_000_000},
		})
	}
	report, err := MustAnalyzer(OperatingScenario()).Analyze(positions, money.Must(500_000))
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultLimitPolicy()
	policy.MaximumRouteShareBps = 2600
	policy.MaximumCustodianBps = 2600
	policy.MaximumOperatorBps = 2600
	policy.MaximumCorridorBps = 2600
	gate, err := EvaluateLimits(report, policy)
	if err != nil {
		t.Fatal(err)
	}
	if gate.Decision != DecisionApprove {
		t.Fatalf("unexpected decision: %+v", gate)
	}
}

func TestScenarioValidationRejectsInvalidBasisPoints(t *testing.T) {
	scenario := OperatingScenario()
	scenario.EscrowHaircutBps = 10_001
	if _, err := NewAnalyzer(scenario); err == nil {
		t.Fatal("expected scenario validation error")
	}
}

func TestProjectionRejectsNilState(t *testing.T) {
	if _, err := ProjectState(nil); err == nil {
		t.Fatal("expected nil state error")
	}
}

func sampleState() *domain.State {
	state := domain.NewState("risk-test")
	routeA := domain.NewRoute("route-a", "usdc", "shipper-a", "operator-a", 1)
	routeA.OriginPort = "esbcn"
	routeA.DestinationPort = "nlrtm"
	routeA.EscrowTotal = 1_000_000
	routeA.EscrowRemaining = 800_000
	routeA.RebateBudget = 40_000
	routeA.RebateClaimed = 10_000
	if err := state.AddRoute(routeA); err != nil {
		panic(err)
	}
	milestoneA := domain.NewMilestone("mile-a", routeA.ID, 1)
	milestoneA.Amount = 1_000_000
	milestoneA.CustodianID = "custody-a"
	milestoneA.ReleasedAmount = 200_000
	if err := state.AddMilestone(milestoneA); err != nil {
		panic(err)
	}

	routeB := domain.NewRoute("route-b", "usdc", "shipper-b", "operator-b", 1)
	routeB.OriginPort = "nlrtm"
	routeB.DestinationPort = "sgsin"
	routeB.EscrowTotal = 1_000_000
	routeB.EscrowRemaining = 1_000_000
	routeB.RebateBudget = 20_000
	if err := state.AddRoute(routeB); err != nil {
		panic(err)
	}
	milestoneB := domain.NewMilestone("mile-b", routeB.ID, 1)
	milestoneB.Amount = 1_000_000
	milestoneB.CustodianID = "custody-b"
	milestoneB.FrozenAmount = 100_000
	if err := state.AddMilestone(milestoneB); err != nil {
		panic(err)
	}
	return state
}
