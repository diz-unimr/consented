package web

import (
	"consented/pkg/config"
	"consented/pkg/consent"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/samply/golang-fhir-models/fhir-models/fhir"
	"net/http"
)

type Server struct {
	config     config.AppConfig
	gicsClient *consent.GicsHttpClient
}

func NewServer(config config.AppConfig) *Server {
	return &Server{
		config:     config,
		gicsClient: consent.NewGicsClient(config),
	}
}

func (s Server) Run() {
	r := s.setupRouter()

	log.Info().Str("port", s.config.App.Http.Port).Msg("Starting server")
	for _, v := range r.Routes() {
		log.Info().Str("path", v.Path).Str("method", v.Method).Msg("Route configured")
	}

	log.Fatal().Err(r.Run(":" + s.config.App.Http.Port)).Msg("Server failed to run")
}

func (s Server) setupRouter() *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(config.DefaultStructuredLogger(), gin.Recovery())

	r.GET("/consent/status/:pid/:domain", gin.BasicAuth(gin.Accounts{
		s.config.App.Http.Auth.User: s.config.App.Http.Auth.Password,
	}), s.handleConsentStatus)

	return r
}

type StatusRequest struct {
	PatientId string  `uri:"pid" binding:"required"`
	Domain    string  `uri:"domain" binding:"required"`
	Date      *string `form:"date"`
}

func (s Server) handleConsentStatus(c *gin.Context) {

	// bind to struct
	var r StatusRequest
	if err := c.ShouldBindUri(&r); err != nil {
		log.Error().Err(err).Msg("Failed to parse request parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	if err := c.ShouldBindQuery(&r); err != nil {
		log.Error().Err(err).Msg("Failed to parse request query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	resp, err, code := s.gicsClient.GetConsentStatus(r.PatientId, r.Domain, r.Date)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get consent status from gICS")
		c.JSON(code, gin.H{
			"error": err.Error(),
		})
		return
	}

	v := getConsented(resp)
	if v == nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Received unexpected response from gICS",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"domain":    r.Domain,
		"consented": *v,
	})
}

func getConsented(p *fhir.Parameters) *bool {
	for _, v := range p.Parameter {
		if v.Name == "consented" {
			return v.ValueBoolean
		}
	}
	return nil
}
