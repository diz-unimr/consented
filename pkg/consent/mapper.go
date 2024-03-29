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
	DocumentRef *string    `json:"document-ref"`
	Status      string     `json:"status"`
	LastUpdated *time.Time `json:"last-updated"`
	AskConsent  bool       `json:"ask-consent"`
	Policies    []Policy   `json:"policies"`
}

type Policy struct {
	Name   string `json:"name"`
	Permit bool   `json:"permit"`
	Code   string `json:"-"`
}

func ParseConsent(b *fhir.Bundle, domain Domain, c GicsClient) (*DomainStatus, error) {

	// fixed max date
	noExpiryDate := time.Date(3000, 1, 1, 0, 0, 0, 0, time.Local)

	// status result
	ds := DomainStatus{
		Domain:      domain.Name,
		Description: domain.Description,
		DocumentRef: domain.DocumentRef,
		LastUpdated: nil,
		AskConsent:  true,
		Status:      Status(NotAsked).String(),
		Policies:    make([]Policy, 0),
	}

	// return if bundle is empty
	if len(b.Entry) == 0 {
		return &ds, nil
	}

	checkPolicyFound := false
	// check consent resources
	for _, e := range b.Entry {
		r, _ := fhir.UnmarshalConsent(e.Resource)

		// last updated
		updated := parseTime(r.DateTime)
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
			expires := parseTime(r.Provision.Provision[0].Period.End)

			ds.AskConsent = expires.Before(now.AddDate(1, 0, 0))

			if p.Permit {
				ds.Status = Status(Accepted).String()
				if expires.Before(now) {
					// already expired
					ds.Status = Status(Expired).String()
				}

			} else {
				// declined
				ds.Status = Status(Declined).String()

				// check withdrawn state
				if noExpiryDate.Equal(expires) && len(domain.WithdrawalUri) > 0 && domain.WithdrawalUri == c.GetSourceReferenceTemplate(*r.SourceReference.Reference) {
					ds.Status = Status(Withdrawn).String()
				}
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

func parsePolicy(prov *fhir.ConsentProvision) (*Policy, error) {
	// check for provision value(s)
	if p := prov.Provision; len(p) > 0 && len(p[0].Code) > 0 && len(p[0].Code[0].Coding) > 0 {
		// take first coding
		co := p[0].Code[0].Coding[0]
		var name string
		if co.Display != nil && strings.TrimSpace(*co.Display) != "" {
			name = strings.TrimSpace(*co.Display)
		} else {
			name = *co.Code
		}

		return &Policy{name, p[0].Type.Code() == fhir.ConsentProvisionTypePermit.Code(), *co.Code}, nil
	}

	return nil, errors.New("missing policy coding")
}

func parseTime(dt *string) time.Time {
	t, err := time.Parse(time.RFC3339, *dt)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse time from RFC3339 string")
		return time.Time{}
	}
	return t
}
