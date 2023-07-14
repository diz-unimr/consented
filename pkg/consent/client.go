package consent

import (
	"bytes"
	"consented/pkg/config"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"io"
	"net/http"
	"time"
)

const DateLayout = "2006-01-02"

type GicsHttpClient struct {
	Auth            *config.Auth
	RequestUrl      string
	ConsentProfiles map[string]*Profile
}

func NewGicsClient(config config.AppConfig) *GicsHttpClient {
	client := &GicsHttpClient{
		RequestUrl:      config.Gics.Fhir.Base + "/$isConsented",
		ConsentProfiles: GetSupportedProfiles(config.Gics.SignerId),
	}
	if config.Gics.Fhir.Auth != nil {
		client.Auth = config.Gics.Fhir.Auth
	}

	return client
}

func (c *GicsHttpClient) GetConsentStatus(signerId, domain string, date *string) (*fhir.Parameters, error, int) {
	// parse date
	consentDate, err := parseDate(date)
	if err != nil {
		return nil, err, http.StatusBadRequest
	}

	// get profile
	p, exists := c.ConsentProfiles[domain]
	if !exists {
		err := fmt.Errorf("domain %s not supported", domain)
		log.Error().Str("domain", domain).Msg("Domain not supported")
		return nil, err, http.StatusNotFound
	}

	// default config
	ignoreVersionNumber := false
	unknownStateIsConsideredAsDecline := true
	configParam, err := fhir.Parameters{
		Parameter: []fhir.ParametersParameter{
			{
				Name:         "ignoreVersionNumber",
				ValueBoolean: &ignoreVersionNumber,
			},
			{
				Name:         "unknownStateIsConsideredAsDecline",
				ValueBoolean: &unknownStateIsConsideredAsDecline,
			},
			{
				Name:      "requestDate",
				ValueDate: &consentDate,
			},
		},
	}.MarshalJSON()
	if err != nil {
		log.Error().Err(err).Msg("Unable to serialize config parameter")
		return nil, err, http.StatusBadRequest
	}

	fhirRequest := fhir.Parameters{
		Id:   nil,
		Meta: nil,
		Parameter: []fhir.ParametersParameter{
			{
				Name:            "personIdentifier",
				ValueIdentifier: &fhir.Identifier{System: p.PersonIdSystem, Value: &signerId},
			},
			{
				Name:        "domain",
				ValueString: &domain,
			},
			{
				Name:        "policy",
				ValueCoding: p.PolicyCoding,
			},
			{
				Name:        "version",
				ValueString: p.PolicyVersion,
			},
			{
				Name:     "config",
				Resource: configParam,
			},
		},
	}
	r, err := fhirRequest.MarshalJSON()
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}

	// post request to gICS
	response, err := c.postRequest(r)

	if err != nil {
		log.Error().Err(err).Msg("POST request to gICS failed for: " + c.RequestUrl)
		return nil, err, http.StatusBadGateway
	}
	defer closeBody(response.Body)

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal().Err(err).Msg("Unable to parse gICS get consent status response")
	}
	if response.StatusCode != http.StatusOK {
		err = errors.New(string(responseData))
		log.Error().Err(err).Int("statusCode", response.StatusCode).Msg("POST request to gICS failed")
		return nil, err, http.StatusBadGateway
	}

	res, err := fhir.UnmarshalParameters(responseData)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to deserialize FHIR response from  gICS. Expected 'Parameters' resource")
		return nil, err, http.StatusBadGateway
	}

	return &res, nil, http.StatusOK
}

func (c *GicsHttpClient) postRequest(body []byte) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, c.RequestUrl,
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

func closeBody(body io.ReadCloser) {
	err := body.Close()
	if err != nil {
		log.Error().Err(err).Msg("Failed to close response body")
	}
}

func parseDate(date *string) (string, error) {
	if date == nil {
		return time.Now().Format(DateLayout), nil
	} else {
		_, err := time.Parse(DateLayout, *date)
		if err == nil {
			return *date, nil
		}
		return "", err
	}
}
