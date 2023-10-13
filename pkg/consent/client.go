package consent

import (
	"bytes"
	"consented/pkg/config"
	"errors"
	"github.com/rs/zerolog/log"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"io"
	"net/http"
)

type GicsClient interface {
	GetDomains() ([]fhir.ResearchStudy, error)
	GetConsentPolicies(signerId string, domain Domain) (*fhir.Bundle, error)
}

type GicsHttpClient struct {
	Auth    *config.Auth
	BaseUrl string
}

func NewGicsClient(config config.AppConfig) *GicsHttpClient {
	client := &GicsHttpClient{
		BaseUrl: config.Gics.Fhir.Base,
	}
	if config.Gics.Fhir.Auth != nil {
		client.Auth = config.Gics.Fhir.Auth
	}

	return client
}

func (c *GicsHttpClient) GetDomains() ([]fhir.ResearchStudy, error) {
	data, err := parseResponse(c.getRequest(c.BaseUrl + "/ResearchStudy"))

	// error handling
	if err != nil {
		return nil, err
	}

	// unmarshal
	bundle, err := fhir.UnmarshalBundle(data)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to deserialize FHIR response from  gICS. Expected 'Bundle' of 'ResearchStudy' for domain request")
		return nil, err
	}

	var domains []fhir.ResearchStudy
	for _, e := range bundle.Entry {
		rs, err := fhir.UnmarshalResearchStudy(e.Resource)
		if err != nil {
			log.Error().Err(err).Msg("Failed to deserialize 'ResearchStudy' from domain request")
			return nil, err
		}

		domains = append(domains, rs)
	}

	return domains, nil
}

func parseResponse(response *http.Response, err error) ([]byte, error) {

	if err != nil {
		return nil, err
	}

	defer closeBody(response.Body)

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse gICS response body")
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		err = errors.New(string(responseData))
		log.Error().Err(err).Int("statusCode", response.StatusCode).Msg("Response status not OK")
		return nil, err
	}

	return responseData, nil
}

func (c *GicsHttpClient) GetConsentPolicies(signerId string, domain Domain) (*fhir.Bundle, error) {

	fhirRequest := fhir.Parameters{
		Id:   nil,
		Meta: nil,
		Parameter: []fhir.ParametersParameter{
			{
				Name:            "personIdentifier",
				ValueIdentifier: &fhir.Identifier{System: &domain.PersonIdSystem, Value: &signerId},
			},
			{
				Name:        "domain",
				ValueString: &domain.Name,
			},
		},
	}
	r, err := fhirRequest.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// post request to gICS
	data, err := parseResponse(c.postRequest(c.BaseUrl+"/$currentPolicyStatesForPerson", r))

	if err != nil {
		log.Error().Err(err).Msg("POST request to gICS failed for: " + c.BaseUrl + "/$currentPolicyStatesForPerson")
		return nil, err
	}

	res, err := fhir.UnmarshalBundle(data)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to deserialize FHIR response from  gICS. Expected 'Parameters' resource")
		return nil, err
	}

	return &res, nil
}

func (c *GicsHttpClient) postRequest(requestUrl string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, requestUrl,
		bytes.NewBuffer(body))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create POST request")
		return nil, err
	}
	req.Header.Set("Content-Type", "application/fhir+json")
	if c.Auth != nil {
		req.SetBasicAuth(c.Auth.User, c.Auth.Password)
	}

	return http.DefaultClient.Do(req)
}

func (c *GicsHttpClient) getRequest(requestUrl string) (*http.Response, error) {
	return c.newRequest(http.MethodGet, requestUrl, nil)
}

func (c *GicsHttpClient) newRequest(method string, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create request")
		return nil, err
	}
	req.Header.Set("Content-Type", "application/fhir+json")
	if c.Auth != nil {
		req.SetBasicAuth(c.Auth.User, c.Auth.Password)
	}

	return http.DefaultClient.Do(req)
}

func closeBody(body io.ReadCloser) {
	err := body.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to close response body")
	}
}
