package consent

import (
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestInitialize(t *testing.T) {
	c := &TestGicsClient{}
	d := NewDomainCache(c, 1*time.Hour)

	// act
	d.Initialize()

	expected := []Domain{
		{
			Name:            "Foo",
			Description:     "Foo Domain",
			CheckPolicyCode: "MDAT_erheben",
			PersonIdSystem:  "https://ths-greifswald.de/fhir/gics/identifiers/Patienten-ID",
		},
		{
			Name:            "Bar",
			Description:     "Bar Domain",
			CheckPolicyCode: "MDAT_erheben",
			PersonIdSystem:  "https://ths-greifswald.de/fhir/gics/identifiers/Patienten-ID",
			Departments:     []string{"bar-dep"},
		}}

	assert.Equal(t, expected, d.Domains)
}

type TestGicsClient struct{}

func (c *TestGicsClient) GetDomains() ([]fhir.ResearchStudy, error) {
	signerId := fhir.Extension{
		Url: ContextIdentifierElementSystem,
		Extension: []fhir.Extension{{
			Url:      "system",
			ValueUri: of("https://ths-greifswald.de/fhir/gics/identifiers/Patienten-ID"),
		}}}

	return []fhir.ResearchStudy{
		{
			Identifier:  []fhir.Identifier{{Value: of("Foo")}},
			Description: of("Foo Domain"),
			Extension: []fhir.Extension{signerId,
				{
					Url: ExternalPropertyElementSystem,
					Extension: []fhir.Extension{
						{Url: "key", ValueString: of("checkPolicy")},
						{Url: "value", ValueString: of("MDAT_erheben")},
					},
				},
			},
		},
		{
			Identifier:  []fhir.Identifier{{Value: of("Bar")}},
			Description: of("Bar Domain"),
			Extension: []fhir.Extension{signerId,
				{
					Url: ExternalPropertyElementSystem,
					Extension: []fhir.Extension{
						{Url: "key", ValueString: of("checkPolicy")},
						{Url: "value", ValueString: of("MDAT_erheben")},
					}},
				{
					Url: ExternalPropertyElementSystem,
					Extension: []fhir.Extension{
						{Url: "key", ValueString: of("departments")},
						{Url: "value", ValueString: of("bar-dep")},
					},
				},
			},
		},
		{
			Identifier:  []fhir.Identifier{{Value: of("MissingCheck")}},
			Description: of("Domain missing check policy"),
			Extension:   []fhir.Extension{signerId},
		},
		{
			Identifier:  []fhir.Identifier{{Value: of("StatusNotActive")}},
			Description: of("Domain status not active"),
			Status:      fhir.ResearchStudyStatusWithdrawn,
			Extension: []fhir.Extension{signerId,
				{
					Url: ExternalPropertyElementSystem,
					Extension: []fhir.Extension{
						{Url: "key", ValueString: of("checkPolicy")},
						{Url: "value", ValueString: of("MDAT_erheben")},
					}}},
		},
		{
			Identifier:  []fhir.Identifier{{Value: of("CheckPolicyMisconfigured")}},
			Description: of("Misspelled checkPolicy on domain"),
			Status:      fhir.ResearchStudyStatusWithdrawn,
			Extension: []fhir.Extension{signerId,
				{
					Url: ExternalPropertyElementSystem,
					Extension: []fhir.Extension{
						{Url: "key", ValueString: of("checkPolicy")},
						{Url: "value", ValueString: of("MDAT###erheben")},
					}}},
		},
	}, nil
}

func (c *TestGicsClient) GetConsentPolicies(_ string, _ Domain) (*fhir.Bundle, error) {

	return &fhir.Bundle{}, nil
}

func (c *TestGicsClient) GetTemplate(_ string, _ string) (string, error) {
	return "", nil
}

func (c *TestGicsClient) GetSourceReferenceTemplate(_ string) string {
	return ""
}
