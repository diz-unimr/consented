package web

import (
	"consented/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

type Server struct {
	config config.AppConfig
}

func NewServer(config config.AppConfig) *Server {
	return &Server{config: config}
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

	r.POST("/consent-status", gin.BasicAuth(gin.Accounts{
		s.config.App.Http.Auth.User: s.config.App.Http.Auth.Password,
	}), s.handleConsentStatus)

	return r
}

type StatusRequest struct {
	PatientId *string `bson:"patientId" json:"patientId"`
	Domain    *string `bson:"domain" json:"domain"`
}

func (s Server) handleConsentStatus(c *gin.Context) {

	// bind to struct
	var r StatusRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		log.Error().Err(err).Msg("Failed to parse request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
}
