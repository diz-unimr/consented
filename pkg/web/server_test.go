package web

import (
	"bytes"
	"consented/pkg/config"
	"consented/pkg/consent"
	"github.com/kinbiko/jsonassert"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type HandlerTestCase struct {
	name           string
	requestUrl     string
	Auth           config.Auth
	body           string
	responseStatus int
	response       string
}

type FilterDomainTestCase struct {
	name   string
	filter []string
	result []consent.Domain
}

var testAuth = config.Auth{
	User:     "test",
	Password: "test",
}

func TestHandleConsentStatus(t *testing.T) {

	cases := []HandlerTestCase{
		{
			name:           "handlerMissingPid",
			requestUrl:     "/consent/status",
			Auth:           testAuth,
			responseStatus: 404,
			response:       `{"error":"404 page not found"}`,
		},
		{
			name:       "handlerUnauthorized",
			requestUrl: "/consent/status/42",
			Auth: config.Auth{
				User:     "wrong",
				Password: "auth",
			},
			responseStatus: 401,
		},
		{
			name:           "handlerSuccess",
			requestUrl:     "/consent/status/42",
			Auth:           testAuth,
			responseStatus: 200,
			response:       `[{"domain":"Test","description":"Test Consent","status":"accepted","last-updated":"<<PRESENCE>>","expires":"<<PRESENCE>>","ask-consent": false,"policies":[{"name": "IDAT_TEST","permit": true}]}]`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler(t, c)
		})
	}
}

func handler(t *testing.T, data HandlerTestCase) {
	// setup config
	c := config.AppConfig{
		App: config.App{
			Http: config.Http{
				Auth: testAuth,
			},
		},
		Gics: config.Gics{
			UpdateInterval: "1h",
		},
	}

	s := NewServer(c)
	s.domainCache = &consent.DomainCache{
		Domains: []consent.Domain{{
			Name:            "Test",
			Description:     "Test Consent",
			CheckPolicyCode: "IDAT_TEST",
			PersonIdSystem:  "https://ths-greifswald.de/fhir/gics/identifiers/Patienten-ID",
		},
		},
		Initialized: true,
	}
	s.gicsClient = &TestGicsClient{}
	r := s.setupRouter()

	var body io.Reader
	if data.body != "" {
		body = bytes.NewReader([]byte(data.body))
	}
	req, _ := http.NewRequest(http.MethodPost, data.requestUrl, body)
	req.SetBasicAuth(data.Auth.User, data.Auth.Password)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	respData, _ := io.ReadAll(w.Body)
	response := string(respData)

	// assert code
	assert.Equal(t, data.responseStatus, w.Code)
	// assert body
	ja := jsonassert.New(t)
	ja.Assertf(response, data.response)
}

func TestFilterDomains(t *testing.T) {
	test := consent.Domain{
		Name:            "Test",
		Description:     "Test Domain",
		CheckPolicyCode: "IDAT_Test",
		PersonIdSystem:  "https://ths-greifswald.de/fhir/gics/identifiers/TestID",
	}

	dep := consent.Domain{
		Name:            "Dep",
		Description:     "Department specific Domain",
		CheckPolicyCode: "IDAT_Test",
		PersonIdSystem:  "https://ths-greifswald.de/fhir/gics/identifiers/TestID",
		Departments:     []string{"dep"},
	}

	s := &Server{}
	s.domainCache = consent.NewDomainCache(nil, -1)
	s.domainCache.Domains = []consent.Domain{test, dep}

	for _, c := range []FilterDomainTestCase{
		{
			name:   "filterDomainsAll",
			filter: []string{"dep"},
			result: s.domainCache.Domains,
		},
		{
			name:   "filterDomainsNoDep",
			result: []consent.Domain{test},
		}} {

		t.Run(c.name, func(t *testing.T) {

			// act
			filtered := s.filterDomains(c.filter)
			assert.Equal(t, c.result, filtered)
		})
	}

	filtered := s.filterDomains([]string{})

	assert.Equal(t, []consent.Domain{test}, filtered)
}

func TestServerRun(t *testing.T) {
	c := config.AppConfig{
		App:  config.App{Http: config.Http{Port: "-1", Auth: testAuth}},
		Gics: config.Gics{UpdateInterval: "1h"},
	}

	s := NewServer(c)
	s.gicsClient = &TestGicsClient{}
	err := s.Run()

	expected := "listen tcp: address -1: invalid port"
	assert.EqualErrorf(t, err, expected, "Error should be: %v, got: %v", expected, err)
}

type TestGicsClient struct{}

func (c *TestGicsClient) GetDomains() ([]fhir.ResearchStudy, error) {
	return []fhir.ResearchStudy{}, nil
}

func (c *TestGicsClient) GetConsentPolicies(_ string, domain consent.Domain) (*fhir.Bundle, error) {
	startTime := of(time.Now().Format(time.RFC3339))
	r := fhir.Consent{
		Meta: &fhir.Meta{LastUpdated: startTime},
		Provision: &fhir.ConsentProvision{
			Type: of(fhir.ConsentProvisionTypePermit),
			Period: &fhir.Period{
				Start: startTime,
				End:   of(time.Now().AddDate(5, 0, 0).Format(time.RFC3339)),
			},
			Code: []fhir.CodeableConcept{{
				Coding: []fhir.Coding{{
					System: of("https://ths-greifswald.de/fhir/CodeSystem/gics/Policy/" + domain.Name),
					Code:   &domain.CheckPolicyCode,
				}},
			}},
		},
	}

	cs, _ := r.MarshalJSON()

	return &fhir.Bundle{
		Entry: []fhir.BundleEntry{{
			Resource: cs,
		}},
	}, nil
}

func of[E any](e E) *E {
	return &e
}
