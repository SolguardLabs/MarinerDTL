package ids

import (
	"fmt"
	"strings"
	"unicode"
)

type ID string

func (id ID) String() string {
	return string(id)
}

func New(kind, raw string) (ID, error) {
	value := normalize(raw)
	if value == "" {
		return "", fmt.Errorf("%s id is required", kind)
	}
	if len(value) < 3 {
		return "", fmt.Errorf("%s id %q is too short", kind, raw)
	}
	if len(value) > 72 {
		return "", fmt.Errorf("%s id %q is too long", kind, raw)
	}
	if strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return "", fmt.Errorf("%s id %q has invalid boundary", kind, raw)
	}
	for _, r := range value {
		if isAllowed(r) {
			continue
		}
		return "", fmt.Errorf("%s id %q contains invalid character %q", kind, raw, r)
	}
	return ID(value), nil
}

func Must(kind, raw string) ID {
	id, err := New(kind, raw)
	if err != nil {
		panic(err)
	}
	return id
}

func normalize(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isAllowed(r rune) bool {
	if unicode.IsLower(r) || unicode.IsDigit(r) {
		return true
	}
	switch r {
	case '-', '_', '.', ':':
		return true
	default:
		return false
	}
}

func NewAccountID(raw string) (ID, error) {
	return New("account", raw)
}

func NewAssetID(raw string) (ID, error) {
	return New("asset", raw)
}

func NewRouteID(raw string) (ID, error) {
	return New("route", raw)
}

func NewMilestoneID(raw string) (ID, error) {
	return New("milestone", raw)
}

func NewCertificateID(raw string) (ID, error) {
	return New("certificate", raw)
}

func NewDisputeID(raw string) (ID, error) {
	return New("dispute", raw)
}

func NewJournalID(raw string) (ID, error) {
	return New("journal", raw)
}

func NewEventID(raw string) (ID, error) {
	return New("event", raw)
}

func NewScenarioID(raw string) (ID, error) {
	return New("scenario", raw)
}
