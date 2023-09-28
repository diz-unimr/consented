package consent

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"strings"
	"time"
)

const (
	SignerIdPrefix                = "https://ths-greifswald.de/fhir/gics/identifiers/"
	ExternalPropertyElementSystem = "https://ths-greifswald.de/fhir/StructureDefinition/gics/ExternalPropertyElement"
)

type Domain struct {
	Name            string
	Description     string
	CheckPolicyCode string
	PersonIdSystem  string
	Departments     []string
}

func (d Domain) String() string {
	return d.Name
}

type DomainCache struct {
	Domains        []Domain
	Client         GicsClient
	UpdateInterval time.Duration
	Initialized    bool
}

func NewDomainCache(c GicsClient, interval time.Duration) *DomainCache {
	return &DomainCache{Client: c, UpdateInterval: interval}
}

func (d *DomainCache) Initialize() chan bool {

	// initial call
	d.updateCache()
	log.Info().Int("domains", len(d.Domains)).Str("update-interval", fmt.Sprintf("%s", d.UpdateInterval)).Msg("Successfully initialized domains. Updating periodically.")

	// init polling
	ticker := time.NewTicker(d.UpdateInterval)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				d.updateCache()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return quit
}

func (d *DomainCache) updateCache() {
	// get domains
	rs, err := d.Client.GetDomains()
	if err != nil {
		log.Error().Err(err).Msg("Failed to update domain cache. Data might be out of date.")
		return
	}

	// build domain structs
	var result []Domain
	for _, s := range rs {
		if s.Status != fhir.ResearchStudyStatusActive {
			continue
		}

		// name & description
		domain := Domain{Name: *s.Identifier[0].Value, Description: *s.Description}

		// external properties
		props := parseExternalProperties(s.Extension)
		if val, ok := props["fhirSafeSignerIdType"]; ok {
			domain.PersonIdSystem = SignerIdPrefix + val
		}
		if val, ok := props["departments"]; ok {
			domain.Departments = strings.Split(val, ",")
		}
		if val, ok := props["checkPolicy"]; ok {
			domain.CheckPolicyCode = val
		} else {
			continue
		}

		result = append(result, domain)
	}

	d.Domains = result
	log.Debug().Int("domains", len(d.Domains)).Msg("Updated domain cache")
}

func parseExternalProperties(ext []fhir.Extension) map[string]string {
	props := make(map[string]string)
	for _, e := range ext {
		if k, v := parseProperty(e); k != nil && v != nil {
			props[*k] = *v
		}
	}

	return props
}

func parseProperty(e fhir.Extension) (*string, *string) {
	if e.Url != ExternalPropertyElementSystem {
		return nil, nil
	}

	var key, value *string
	for _, ee := range e.Extension {
		if ee.Url == "key" {
			key = ee.ValueString
		} else if ee.Url == "value" {
			value = ee.ValueString
		}
	}

	return key, value
}

func Of[E any](e E) *E {
	return &e
}
