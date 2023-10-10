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
	actual, _, _ := c.GetConsentPolicies("bla", Domain{
		Name:            "Foo",
		Description:     "Bar",
		CheckPolicyCode: "123",
		PersonIdSystem:  "test",
	})

	assert.Equal(t, expected, actual)
}

func withTestServer(response []byte, code int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

		res.WriteHeader(code)
		_, _ = res.Write(response)
	}))
}
