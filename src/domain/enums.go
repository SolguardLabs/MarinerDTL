package domain

import "strings"

type AccountRole string

const (
	RoleShipper   AccountRole = "shipper"
	RoleCarrier   AccountRole = "carrier"
	RoleOperator  AccountRole = "operator"
	RoleCustodian AccountRole = "custodian"
	RoleTreasury  AccountRole = "treasury"
	RoleInsurer   AccountRole = "insurer"
	RoleAuditor   AccountRole = "auditor"
)

func AccountRoleFromString(raw string) AccountRole {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "shipper", "customer", "merchant":
		return RoleShipper
	case "carrier", "line", "hauler":
		return RoleCarrier
	case "operator", "ops":
		return RoleOperator
	case "custodian", "surveyor", "agent":
		return RoleCustodian
	case "treasury", "sponsor":
		return RoleTreasury
	case "insurer":
		return RoleInsurer
	case "auditor":
		return RoleAuditor
	default:
		return RoleShipper
	}
}

type RouteStatus string

const (
	RoutePlanned   RouteStatus = "planned"
	RouteFunded    RouteStatus = "funded"
	RouteInTransit RouteStatus = "in_transit"
	RouteCompleted RouteStatus = "completed"
	RouteCancelled RouteStatus = "cancelled"
	RouteSuspended RouteStatus = "suspended"
)

type MilestoneStatus string

const (
	MilestonePlanned   MilestoneStatus = "planned"
	MilestoneLocked    MilestoneStatus = "locked"
	MilestoneCertified MilestoneStatus = "certified"
	MilestoneReleased  MilestoneStatus = "released"
	MilestoneDisputed  MilestoneStatus = "disputed"
	MilestoneCancelled MilestoneStatus = "cancelled"
)

type DisputeStatus string

const (
	DisputeOpen      DisputeStatus = "open"
	DisputeAccepted  DisputeStatus = "accepted"
	DisputeRejected  DisputeStatus = "rejected"
	DisputeResolved  DisputeStatus = "resolved"
	DisputeCancelled DisputeStatus = "cancelled"
)

type CertificateStatus string

const (
	CertificateIssued   CertificateStatus = "issued"
	CertificateVerified CertificateStatus = "verified"
	CertificateRevoked  CertificateStatus = "revoked"
)

type EventType string

const (
	EventAccountRegistered  EventType = "account_registered"
	EventAssetRegistered    EventType = "asset_registered"
	EventDeposit            EventType = "deposit"
	EventRouteCreated       EventType = "route_created"
	EventEscrowFunded       EventType = "escrow_funded"
	EventMilestoneAdded     EventType = "milestone_added"
	EventCertificateIssued  EventType = "certificate_issued"
	EventMilestoneReleased  EventType = "milestone_released"
	EventDisputeOpened      EventType = "dispute_opened"
	EventDisputeResolved    EventType = "dispute_resolved"
	EventMilestoneCancelled EventType = "milestone_cancelled"
	EventRouteCancelled     EventType = "route_cancelled"
	EventRebateClaimed      EventType = "rebate_claimed"
	EventAudit              EventType = "audit"
)
