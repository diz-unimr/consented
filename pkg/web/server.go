package web

import (
	"consented/pkg/config"
	"consented/pkg/consent"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"slices"
	"time"
)

type Server struct {
	config      config.AppConfig
	gicsClient  consent.GicsClient
	domainCache *consent.DomainCache
}

func NewServer(config config.AppConfig) *Server {
	c := consent.NewGicsClient(config)
	interval, err := time.ParseDuration(config.Gics.UpdateInterval)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse 'gics.update-interval' from app config")
		os.Exit(1)
	}

	return &Server{
		config:      config,
		gicsClient:  c,
		domainCache: consent.NewDomainCache(c, interval),
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
	r.GET("/health", s.checkHealth)
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

func (s *Server) handleConsentStatus(c *gin.Context) {

	// bind to struct
	var r StatusRequest
	// path parameter is matched by route
	_ = c.ShouldBindUri(&r)
	// body is optional
	_ = c.ShouldBindJSON(&r)

	response := make([]consent.DomainStatus, 0)
	// filter domains by department
	for _, d := range s.filterDomains(r.Departments) {

		// get status per domain
		ds, err := s.createDomainStatus(r, d)
		if err != nil {
			continue
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

func (s *Server) createDomainStatus(r StatusRequest, d consent.Domain) (*consent.DomainStatus, error) {
	// get current policies
	resp, err := s.gicsClient.GetConsentPolicies(r.PatientId, d)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get consent status from gICS")
		return nil, err
	}

	// parse resources
	ds, err := consent.ParseConsent(resp, d, s.gicsClient)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse consent policies from gICS")
		return nil, err
	}

	return ds, nil
}

func (s *Server) checkHealth(c *gin.Context) {
	if s.domainCache.IsHealthy {
		c.JSON(http.StatusOK, gin.H{
			"healthy": true,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"healthy": false,
		})
	}
}
