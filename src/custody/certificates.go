package custody

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"marinerdtl/src/domain"
	"marinerdtl/src/policy"
)

type Registry struct {
	policy policy.Policy
	state  *domain.State
}

func NewRegistry(state *domain.State, rules policy.Policy) Registry {
	return Registry{state: state, policy: rules}
}

func (r Registry) ValidateCustodian(accountID string) (*domain.Account, error) {
	account, err := r.state.RequireAccount(accountID)
	if err != nil {
		return nil, err
	}
	if r.policy.RequireCustodianRole && account.Role != domain.RoleCustodian {
		return nil, fmt.Errorf("account %s is not an approved custodian", account.ID)
	}
	return account, nil
}

func (r Registry) ValidateCarrier(accountID string) (*domain.Account, error) {
	account, err := r.state.RequireAccount(accountID)
	if err != nil {
		return nil, err
	}
	if account.Role != domain.RoleCarrier && account.Role != domain.RoleOperator {
		return nil, fmt.Errorf("account %s cannot receive carrier settlement", account.ID)
	}
	return account, nil
}

func (r Registry) ValidateCertificateDraft(route *domain.Route, milestone *domain.Milestone, custodianID, documentHash string) error {
	if route == nil || milestone == nil {
		return fmt.Errorf("route and milestone are required")
	}
	if milestone.RouteID != route.ID {
		return fmt.Errorf("milestone %s does not belong to route %s", milestone.ID, route.ID)
	}
	if milestone.CustodianID != custodianID {
		return fmt.Errorf("custodian %s is not assigned to milestone %s", custodianID, milestone.ID)
	}
	if !milestone.CanCertify() {
		return fmt.Errorf("milestone %s cannot receive certificate in status %s", milestone.ID, milestone.Status)
	}
	if _, err := r.ValidateCustodian(custodianID); err != nil {
		return err
	}
	return r.policy.CheckCertificateHash(documentHash)
}

func (r Registry) VerifyCertificate(certificate *domain.Certificate, route *domain.Route, milestone *domain.Milestone) error {
	if certificate == nil {
		return fmt.Errorf("certificate is required")
	}
	if certificate.RouteID != route.ID {
		return fmt.Errorf("certificate route mismatch")
	}
	if certificate.MilestoneID != milestone.ID {
		return fmt.Errorf("certificate milestone mismatch")
	}
	if certificate.CustodianID != milestone.CustodianID {
		return fmt.Errorf("certificate custodian mismatch")
	}
	if certificate.Status == domain.CertificateRevoked {
		return fmt.Errorf("certificate %s is revoked", certificate.ID)
	}
	return r.policy.CheckCertificateHash(certificate.DocumentHash)
}

func Fingerprint(parts ...string) string {
	joined := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(sum[:])
}

func NormalizeHash(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func HasCertificateFor(state *domain.State, milestoneID string) bool {
	for _, certificate := range state.Certificates {
		if certificate.MilestoneID == milestoneID && certificate.Status != domain.CertificateRevoked {
			return true
		}
	}
	return false
}

func CertificateChain(state *domain.State, routeID string) []domain.Certificate {
	out := []domain.Certificate{}
	for _, id := range state.SortedCertificateIDs() {
		certificate := state.Certificates[id]
		if certificate.RouteID == routeID {
			out = append(out, *certificate.Clone())
		}
	}
	return out
}
