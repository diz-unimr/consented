package consent

import (
	"consented/pkg/config"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetDomains(t *testing.T) {

	res, _ := fhir.ResearchStudy{}.MarshalJSON()
	b, _ := fhir.Bundle{
		Entry: []fhir.BundleEntry{{Resource: res}}}.MarshalJSON()

	s := withTestServer(b, 200)
	defer s.Close()

	c := NewGicsClient(config.AppConfig{Gics: config.Gics{
		UpdateInterval: "1h", Fhir: config.Fhir{Base: s.URL},
	}})

	actual, _ := c.GetDomains()
	expected := []fhir.ResearchStudy{{}}

	assert.Equal(t, expected, actual)
}

func TestGetConsentPolicies(t *testing.T) {

	rs, _ := fhir.ResearchStudy{Id: of("42")}.MarshalJSON()
	expected := &fhir.Bundle{Entry: []fhir.BundleEntry{
		{Resource: rs},
	}}

	resp, _ := expected.MarshalJSON()
	s := withTestServer(resp, 200)
	defer s.Close()

	c := NewGicsClient(config.AppConfig{Gics: config.Gics{
		UpdateInterval: "1h", Fhir: config.Fhir{Base: s.URL, Auth: &config.Auth{
			User:     "test",
			Password: "test",
		}},
	}})

	// act
	actual, _ := c.GetConsentPolicies("bla", Domain{
		Name:            "Foo",
		Description:     "Bar",
		CheckPolicyCode: "123",
		PersonIdSystem:  "test",
	})

	assert.Equal(t, expected, actual)
}

func TestGetSourceReferenceTemplate(t *testing.T) {
	templateUri := "Widerruf+%28kompatibel+zu+Patienteneinwilligung+MII+1.6d%29|2.0.a"

	resp, _ := fhir.QuestionnaireResponse{
		Questionnaire: of("https://ths-greifswald.de/fhir/gics/QuestionnaireComposed/MII/" + templateUri),
	}.MarshalJSON()

	s := withTestServer(resp, 200)
	defer s.Close()

	c := NewGicsClient(config.AppConfig{Gics: config.Gics{
		Fhir: config.Fhir{Base: s.URL + "/"},
	}})

	template := c.GetSourceReferenceTemplate("QuestionnaireResponse/42")

	assert.Equal(t, template, templateUri)
}

func TestGetTemplate(t *testing.T) {
	expected := "Widerruf+%28kompatibel+zu+Patienteneinwilligung+MII+1.6d%29|2.0.a"
	qs, _ := fhir.Questionnaire{
		Code: []fhir.Coding{{
			System: of("http://fhir.de/ConsentManagement/CodeSystem/TemplateType"),
			Code:   of("WITHDRAWAL"),
		}},
		Url: of("https://ths-greifswald.de/fhir/gics/ConsentTemplate/MII/Widerruf+%28kompatibel+zu+Patienteneinwilligung+MII+1.6d%29|2.0.a"),
	}.MarshalJSON()
	b := &fhir.Bundle{Entry: []fhir.BundleEntry{
		{Resource: qs},
	}}
	resp, _ := b.MarshalJSON()

	s := withTestServer(resp, 200)
	defer s.Close()

	c := NewGicsClient(config.AppConfig{Gics: config.Gics{
		Fhir: config.Fhir{Base: s.URL + "/"},
	}})

	actual := c.GetTemplate("Test", "WITHDRAWAL")

	assert.Equal(t, expected, actual)
}

func TestGetTemplate_WithErrors(t *testing.T) {

	cases := []struct {
		name string
		data string
		code int
	}{
		{
			name: "errorResponse",
			code: 400,
		},
		{
			name: "invalidFhir",
			data: "<invalid-fhir>",
			code: 200,
		},
		{
			name: "invalidFhir",
			data: "{\"resourceType\": \"Bundle\", \"entry\": [{\"bla\":\"blubb\"}]}",
			code: 200,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			s := withTestServer([]byte(c.data), c.code)
			defer s.Close()
			c := NewGicsClient(config.AppConfig{Gics: config.Gics{
				Fhir: config.Fhir{Base: s.URL + "/"},
			}})

			actual := c.GetTemplate("Test", "WITHDRAWAL")
			assert.Equal(t, "", actual)
		})
	}
}

func withTestServer(response []byte, code int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		res.WriteHeader(code)
		_, _ = res.Write(response)
	}))
}
