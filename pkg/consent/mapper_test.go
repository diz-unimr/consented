package consent

import (
	"errors"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type Expected struct {
	result *DomainStatus
	error  error
}

type ParseConsentTestCase struct {
	name     string
	domain   Domain
	policies []fhir.Consent
	expected Expected
}

func TestParseConsent(t *testing.T) {
	// prepare dates
	now := time.Now().UTC()
	now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(),
		now.Minute(), now.Second(), 0, now.Location())

	// test cases
	cases := []ParseConsentTestCase{
		{
			name: "parseConsent_Success",
			domain: Domain{
				Name:            "Test",
				Description:     "Test domain",
				CheckPolicyCode: "MDAT_erheben",
				PersonIdSystem:  "Patient-ID",
			},
			// policy consent resources
			policies: getTestConsentPolicies(now),
			expected: Expected{
				&DomainStatus{
					Domain:      "Test",
					Description: "Test domain",
					Status:      "accepted",
					LastUpdated: &now,
					Expires:     of(now.AddDate(5, 0, 0)),
					AskConsent:  false,
					Policies: []Policy{
						{"MDAT_erheben", true, "MDAT_erheben"},
						{"MDAT_speichern_verarbeiten", false, "MDAT_speichern_verarbeiten"},
					},
				}, nil,
			},
		},
		{
			name: "parseConsent_FailsWithInvalidCheckPolicy",
			domain: Domain{
				Name:            "Test",
				Description:     "Test domain",
				CheckPolicyCode: "#MDAT_erheben#",
				PersonIdSystem:  "Patient-ID",
			},
			// policy consent resources
			policies: getTestConsentPolicies(now),
			expected: Expected{
				nil,
				errors.New("checkPolicy not found for domain"),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parseConsentHandler(t, c)
		})
	}
}

func getTestConsentPolicies(from time.Time) []fhir.Consent {
	return []fhir.Consent{
		{
			Meta: &fhir.Meta{
				LastUpdated: of(from.Format(time.RFC3339)),
			},
			Provision: of(fhir.ConsentProvision{
				Type: of(fhir.ConsentProvisionTypePermit),
				Period: &fhir.Period{
					Start: of(from.Format(time.RFC3339)),
					End:   of(from.AddDate(5, 0, 0).Format(time.RFC3339)),
				},
				Code: []fhir.CodeableConcept{{
					Coding: []fhir.Coding{{
						System: of("https://ths-greifswald.de/fhir/CodeSystem/gics/Policy/MII"),
						Code:   of("MDAT_erheben"),
					}},
				}},
			}),
		},
		{
			Meta: &fhir.Meta{
				LastUpdated: of(from.Format(time.RFC3339)),
			},
			Provision: of(fhir.ConsentProvision{
				Type: of(fhir.ConsentProvisionTypeDeny),
				Period: &fhir.Period{
					Start: of(from.Format(time.RFC3339)),
					End:   of(from.AddDate(10, 0, 0).Format(time.RFC3339)),
				},
				Code: []fhir.CodeableConcept{{
					Coding: []fhir.Coding{{
						System: of("https://ths-greifswald.de/fhir/CodeSystem/gics/Policy/MII"),
						Code:   of("MDAT_speichern_verarbeiten"),
					}},
				}},
			}),
		},
	}
}

func parseConsentHandler(t *testing.T, c ParseConsentTestCase) {

	entries := make([]fhir.BundleEntry, 0)
	for _, p := range c.policies {
		r, _ := p.MarshalJSON()
		entries = append(entries, fhir.BundleEntry{Resource: r})
	}

	bundle := &fhir.Bundle{Entry: entries}

	// act
	res, err := ParseConsent(bundle, c.domain)

	assert.Equal(t, c.expected.result, res)
	assert.Equal(t, c.expected.error, err)
}

type ParsePolicyTestCase struct {
	name     string
	code     string
	display  string
	permit   bool
	expected Policy
}

func TestParsePolicy(t *testing.T) {

	cases := []ParsePolicyTestCase{
		{name: "TestParsePolicyWithDisplay",
			code:    "code",
			display: "display",
			permit:  true,
			expected: Policy{
				Name:   "display",
				Permit: true,
				Code:   "code",
			},
		},

		{name: "TestParsePolicyWithCode",
			code:   "code",
			permit: false,
			expected: Policy{
				Name:   "code",
				Permit: false,
				Code:   "code",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			now := time.Now()
			ct := fhir.ConsentProvisionTypeDeny
			if c.permit {
				ct = fhir.ConsentProvisionTypePermit
			}
			p := fhir.ConsentProvision{
				Type: &ct,
				Period: &fhir.Period{
					Start: of(now.Format(time.RFC3339)),
					End:   of(now.AddDate(5, 0, 0).Format(time.RFC3339)),
				},
				Code: []fhir.CodeableConcept{{
					Coding: []fhir.Coding{{
						Code:    &c.code,
						Display: &c.display,
					}},
				}},
			}

			actual, err := parsePolicy(&p)

			assert.Nil(t, err)
			assert.Equal(t, c.expected, *actual)
		})
	}
}

func of[E any](e E) *E {
	return &e
}
