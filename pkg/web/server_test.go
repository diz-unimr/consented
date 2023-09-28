package web

import (
	"consented/pkg/config"
	"consented/pkg/consent"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestCase struct {
	name           string
	requestUrl     string
	Auth           config.Auth
	responseStatus int
	response       string
}

var testAuth = config.Auth{
	User:     "test",
	Password: "test",
}

func TestHandleConsentStatus(t *testing.T) {

	cases := []TestCase{
		{"handlerMissingPid", "/consent/status", testAuth, 404, "{\"error\":\"404 page not found\"}"},
		{"handlerMissingDomain", "/consent/status/42", testAuth, 404, "{\"error\":\"404 page not found\"}"},
		{"handlerEmptyParameters", "/consent/status//MII", testAuth, 400, "{\"error\":\"validation failed on field 'PatientId', condition: required\"}"},
		{"handlerUnauthorized", "/consent/status/42/MII", config.Auth{
			User:     "wrong",
			Password: "auth",
		}, 401, ""},
		{"handlerSuccess", "/consent/status/42/MII", testAuth, 200, "{\"consented\":true,\"domain\":\"MII\"}"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler(t, c)
		})
	}
}

func handler(t *testing.T, data TestCase) {
	// setup config
	c := config.AppConfig{
		App: config.App{
			Http: config.Http{
				Auth: testAuth,
			},
		},
		Gics: config.Gics{SignerId: "test"},
	}

	s := NewServer(c)
	s.gicsClient = &TestGicsClient{}
	r := s.setupRouter()

	req, _ := http.NewRequest(http.MethodGet, data.requestUrl, nil)
	req.SetBasicAuth(data.Auth.User, data.Auth.Password)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	respData, _ := io.ReadAll(w.Body)
	response := string(respData)

	assert.Equal(t, data.responseStatus, w.Code)
	assert.Equal(t, data.response, response)
}

type TestGicsClient struct{}

func (c *TestGicsClient) GetConsentStatus(_ string, _ consent.Domain, _ *string) (*fhir.Parameters, error, int) {
	isConsented := true
	return &fhir.Parameters{
		Parameter: []fhir.ParametersParameter{{
			Name:         "consented",
			ValueBoolean: &isConsented,
		}},
	}, nil, http.StatusOK
}

func (c *TestGicsClient) GetDomains() ([]*fhir.ResearchStudy, error) {
	return []*fhir.ResearchStudy{}, nil
}
