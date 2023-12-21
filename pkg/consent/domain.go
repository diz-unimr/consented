package consent

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"strings"
	"time"
)

const (
	ContextIdentifierElementSystem = "http://fhir.de/ConsentManagement/StructureDefinition/ContextIdentifier"
	ExternalPropertyElementSystem  = "https://ths-greifswald.de/fhir/StructureDefinition/gics/ExternalPropertyElement"
	TemplateType                   = "http://fhir.de/ConsentManagement/CodeSystem/TemplateType"
)

type Domain struct {
	Name            string
	Description     string
	CheckPolicyCode string
	PersonIdSystem  string
	Departments     []string
	WithdrawalUri   string
}

func (d Domain) String() string {
	return d.Name
}

type DomainCache struct {
	Domains        []Domain
	Client         GicsClient
	UpdateInterval time.Duration
	Initialized    bool
	IsHealthy      bool
}

func NewDomainCache(c GicsClient, interval time.Duration) *DomainCache {
	return &DomainCache{Client: c, UpdateInterval: interval}
}

func (d *DomainCache) Initialize() chan bool {

	// initial call
	d.IsHealthy = d.updateCache()
	log.Info().Int("domains", len(d.Domains)).Str("update-interval", d.UpdateInterval.String()).
		Msg("Successfully initialized domains. Updating periodically.")

	// init polling
	ticker := time.NewTicker(d.UpdateInterval)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				d.IsHealthy = d.updateCache()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return quit
}

func (d *DomainCache) updateCache() bool {
	// get domains
	rs, err := d.Client.GetDomains()
	if err != nil {
		log.Error().Err(err).Msg("Failed to update domain cache. Data might be out of date.")
		return false
	}

	// build domain structs
	var result []Domain
	for _, s := range rs {
		if s.Status != fhir.ResearchStudyStatusActive {
			continue
		}

		name := *s.Identifier[0].Value
		desc := name
		if s.Description != nil {
			desc = *s.Description
		}
		// name & description
		domain := Domain{Name: name, Description: desc}

		// parse id system
		ctxId := parseIdSystem(s.Extension)
		if ctxId != nil {
			domain.PersonIdSystem = *ctxId
		} else {
			continue
		}

		// external properties
		props := parseExternalProperty(s.Extension)
		if val, ok := props["departments"]; ok {
			domain.Departments = strings.Split(val, ",")
		}
		if val, ok := props["checkPolicy"]; ok {
			domain.CheckPolicyCode = val
		} else {
			continue
		}

		if t, e := d.Client.GetTemplate(domain.Name, "WITHDRAWAL"); e == nil {
			domain.WithdrawalUri = t
		}

		result = append(result, domain)
	}

	d.Domains = result
	log.Debug().Str("domains", fmt.Sprintf("%s", d.Domains)).Msg("Updated domain cache")
	return true
}

func parseIdSystem(ext []fhir.Extension) *string {
	for _, e := range ext {
		if e.Url != ContextIdentifierElementSystem {
			continue
		}

		for _, ee := range e.Extension {
			if ee.Url == "system" {
				return ee.ValueUri
			}
		}
	}
	return nil
}

func parseExternalProperty(ext []fhir.Extension) map[string]string {
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
