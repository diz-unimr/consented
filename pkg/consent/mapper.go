package consent

import (
	"errors"
	"github.com/rs/zerolog/log"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"strings"
	"time"
)

type DomainStatus struct {
	Domain      string     `json:"domain"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	LastUpdated *time.Time `json:"last-updated"`
	Expires     *time.Time `json:"expires"`
	AskConsent  bool       `json:"ask-consent"`
	Policies    []Policy   `json:"policies"`
}

type Policy struct {
	Name   string `json:"name"`
	Permit bool   `json:"permit"`
	Code   string `json:"-"`
}

func ParseConsent(b *fhir.Bundle, domain Domain) (*DomainStatus, error) {

	// fixed max date
	noExpiryDate := time.Date(3000, 1, 1, 0, 0, 0, 0, time.Local)

	// status result
	ds := DomainStatus{
		Domain:      domain.Name,
		Description: domain.Description,
		LastUpdated: nil,
		AskConsent:  true,
		Status:      Status(NotAsked).String(),
		Policies:    make([]Policy, 0),
	}

	checkPolicyFound := false
	// check consent resources
	for _, e := range b.Entry {
		r, _ := fhir.UnmarshalConsent(e.Resource)

		// last updated
		updated := parseTime(r.Meta.LastUpdated)
		if ds.LastUpdated == nil || updated.After(*ds.LastUpdated) {
			ds.LastUpdated = &updated
		}

		// policy
		p, err := parsePolicy(r.Provision)
		if err != nil {
			log.Error().Err(err).Msg("Unable to parse policy from Consent resource")
			return nil, err
		}
		ds.Policies = append(ds.Policies, *p)

		// status policy & expiration
		now := time.Now()
		if p.Code == domain.CheckPolicyCode {
			checkPolicyFound = true
			expires := parseTime(r.Provision.Period.End)
			if expires == noExpiryDate {
				ds.Expires = nil
			} else {
				ds.Expires = &expires
			}

			ds.AskConsent = expires.Before(now.AddDate(1, 0, 0))

			if p.Permit {
				ds.Status = Status(Accepted).String()
				if expires.Before(now) {
					// already expired
					ds.Status = Status(Expired).String()
				}

			} else {
				ds.Status = Status(Declined).String()
			}
		}
	}

	// checkPolicy not found
	if !checkPolicyFound {
		log.Error().
			Str("domain", domain.Name).
			Str("checkPolicy", domain.CheckPolicyCode).
			Msg("Unable to determine consent status. Configured policy not found")
		return nil, errors.New("checkPolicy not found for domain")
	}

	return &ds, nil
}

func parsePolicy(p *fhir.ConsentProvision) (*Policy, error) {
	if len(p.Code) > 0 && len(p.Code[0].Coding) > 0 {
		// take first
		co := p.Code[0].Coding[0]
		var name string
		if co.Display != nil && strings.TrimSpace(*co.Display) != "" {
			name = strings.TrimSpace(*co.Display)
		} else {
			name = *co.Code
		}

		return &Policy{name, p.Type.Code() == fhir.ConsentProvisionTypePermit.Code(), *co.Code}, nil
	}

	return nil, errors.New("missing policy coding")
}

func parseTime(dt *string) time.Time {
	t, err := time.Parse(time.RFC3339, *dt)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse lastUpdated from Consent resource")
		return time.Time{}
	}
	return t
}
