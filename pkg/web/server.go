package web

import (
	"consented/pkg/config"
	"consented/pkg/consent"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"net/http"
	"os"
	"slices"
	"time"
)

type Server struct {
	config       config.AppConfig
	gicsClient   consent.GicsClient
	domainCache  *consent.DomainCache
	noExpiryDate time.Time
}

func NewServer(config config.AppConfig) *Server {
	c := consent.NewGicsClient(config)
	interval, err := time.ParseDuration(config.Gics.UpdateInterval)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse 'gics.update-interval' from app config")
		os.Exit(1)
	}

	return &Server{
		config:       config,
		gicsClient:   c,
		domainCache:  consent.NewDomainCache(c, interval),
		noExpiryDate: time.Date(3000, 1, 1, 0, 0, 0, 0, time.Local),
	}
}

func (s *Server) Run() error {
	s.Init()
	r := s.setupRouter()

	log.Info().Str("port", s.config.App.Http.Port).Msg("Starting server")
	for _, v := range r.Routes() {
		log.Info().Str("path", v.Path).Str("method", v.Method).Msg("Route configured")
	}

	return r.Run(":" + s.config.App.Http.Port)
}

func (s *Server) setupRouter() *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(config.DefaultStructuredLogger(), gin.Recovery())

	r.POST("/consent/status/:pid", gin.BasicAuth(gin.Accounts{
		s.config.App.Http.Auth.User: s.config.App.Http.Auth.Password,
	}), s.handleConsentStatus)
	r.NoRoute(gin.BasicAuth(gin.Accounts{
		s.config.App.Http.Auth.User: s.config.App.Http.Auth.Password,
	}), func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "404 page not found"})
	})

	return r
}

type StatusRequest struct {
	PatientId   string   `uri:"pid" binding:"required"`
	Departments []string `json:"departments"`
}

type DomainStatus struct {
	Domain      string     `json:"domain"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	LastUpdated *time.Time `json:"last-updated"`
	Expires     *time.Time `json:"expires"`
	AskConsent  bool       `json:"ask-consent"`
	Policies    []Policy   `json:"policies"`
}

type Policy struct {
	Name   string `json:"name"`
	Permit bool   `json:"permit"`
	Code   string `json:"-"`
}

func (s *Server) handleConsentStatus(c *gin.Context) {

	// bind to struct
	var r StatusRequest
	// path parameter is matched by route
	_ = c.ShouldBindUri(&r)
	// body is optional
	_ = c.ShouldBindJSON(&r)

	var response []DomainStatus
	// filter domains by department
	for _, d := range s.filterDomains(r.Departments) {

		// get status per domain
		ds, err, code := s.createDomainStatus(r, d)
		if err != nil {
			c.JSON(code, gin.H{
				"error": err.Error(),
			})
			return
		}

		response = append(response, *ds)
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) Init() {
	s.domainCache.Initialize()
}

func (s *Server) filterDomains(deps []string) []consent.Domain {
	var domains []consent.Domain
	for _, d := range s.domainCache.Domains {
		// no restrictions
		if len(d.Departments) > 0 {
			for _, required := range d.Departments {
				if slices.Contains(deps, required) {
					domains = append(domains, d)
					break
				}
			}
			continue
		}
		domains = append(domains, d)
	}

	return domains
}

func (s *Server) createDomainStatus(r StatusRequest, d consent.Domain) (*DomainStatus, error, int) {
	// get current policies
	resp, err, code := s.gicsClient.GetConsentPolicies(r.PatientId, d)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get consent status from gICS")
		return nil, err, code
	}

	// parse resources
	ds, err := s.parseConsent(resp, d)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse consent policies from gICS")
		return nil, err, http.StatusInternalServerError
	}

	return ds, nil, http.StatusOK
}

func (s *Server) parseConsent(b *fhir.Bundle, domain consent.Domain) (*DomainStatus, error) {

	// status result
	ds := DomainStatus{
		Domain:      domain.Name,
		Description: domain.Description,
		LastUpdated: nil,
		AskConsent:  true,
		Status:      consent.Status(consent.NotAsked).String(),
		Policies:    make([]Policy, 0),
	}

	// check consent resources
	for _, e := range b.Entry {
		r, _ := fhir.UnmarshalConsent(e.Resource)

		// last updated
		updated := parseTime(r.Meta.LastUpdated)
		if ds.LastUpdated == nil || updated.After(*ds.LastUpdated) {
			ds.LastUpdated = &updated
		}

		// policy
		p, err := parsePolicy(r.Provision)
		if err != nil {
			log.Error().Err(err).Msg("Unable to parse policy from Consent resource")
			return nil, err
		}
		ds.Policies = append(ds.Policies, *p)

		// status policy & expiration
		now := time.Now()
		if p.Code == domain.CheckPolicyCode {
			expires := parseTime(r.Provision.Period.End)
			if expires == s.noExpiryDate {
				ds.Expires = nil
			} else {
				ds.Expires = &expires
			}

			ds.AskConsent = expires.Before(now.AddDate(1, 0, 0))

			if p.Permit {
				ds.Status = consent.Status(consent.Accepted).String()
				if expires.Before(now) {
					// already expired
					ds.Status = consent.Status(consent.Expired).String()
				}

			} else {
				ds.Status = consent.Status(consent.Declined).String()
			}
		}
	}

	return &ds, nil
}

func parsePolicy(p *fhir.ConsentProvision) (*Policy, error) {
	if len(p.Code) > 0 && len(p.Code[0].Coding) > 0 {
		// take first
		co := p.Code[0].Coding[0]
		var name string
		if co.Display != nil {
			name = *co.Display
		} else {
			name = *co.Code
		}

		return &Policy{name, p.Type.Code() == fhir.ConsentProvisionTypePermit.Code(), *co.Code}, nil
	}

	return nil, errors.New("missing policy coding")
}

func parseTime(dt *string) time.Time {
	t, err := time.Parse(time.RFC3339, *dt)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse lastUpdated from Consent resource")
		return time.Time{}
	}
	return t
}
