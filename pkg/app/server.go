package app

import (
	"fmt"
	"log"
	"net/http"

	"github.com/yourorg/github-code-agent/features/analyzer"
	"github.com/yourorg/github-code-agent/features/fixer"
	"github.com/yourorg/github-code-agent/features/gitops"
	"github.com/yourorg/github-code-agent/features/reviewer"
	"github.com/yourorg/github-code-agent/features/standards"
	"github.com/yourorg/github-code-agent/features/webhook"
	"github.com/yourorg/github-code-agent/pkg/constants"
	ctxpkg "github.com/yourorg/github-code-agent/pkg/context"
)

// Server manages the HTTP server and routing
type Server struct {
	container *Container
	handler   http.Handler
}

// NewServer creates a new server instance
func NewServer(container *Container) *Server {
	return &Server{
		container: container,
	}
}

// RegisterAgents registers all feature reasoners and skills
func (s *Server) RegisterAgents() {
	log.Println("Registering reasoners and skills...")

	app := s.container.Agent
	cfg := s.container.Config
	ghClient := s.container.GitHubClient.GetClient()

	// Register AI-powered reasoners
	log.Println("  - Registering reasoners (AI-powered functions)...")
	reviewer.RegisterReasoners(app)
	fixer.RegisterReasoners(app)
	webhook.RegisterReasoners(app)
	analyzer.RegisterReasoners(app)
	standards.RegisterReasoners(app, cfg)
	gitops.RegisterReasoners(app, ghClient)

	// Register deterministic skills
	log.Println("  - Registering skills (deterministic functions)...")
	webhook.RegisterSkills(app)
	analyzer.RegisterSkills(app)
	fixer.RegisterSkills(app)
	standards.RegisterSkills(app, cfg)
	gitops.RegisterSkills(app, ghClient)
}

// SetupRoutes sets up HTTP routing
func (s *Server) SetupRoutes() {
	// Create webhook handler
	webhookHandler := webhook.NewServer(
		s.container.Agent,
		s.container.EnvConfig.GitHubWebhookSecret,
		s.container.GitHubClient,
		s.container.Config,
	)

	// Create AgentField handler
	agentFieldHandler := s.container.Agent.Handler()

	// Create main router
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add dependencies to context
		ctx := ctxpkg.WithAll(
			r.Context(),
			s.container.Agent,
			s.container.Config,
			s.container.GitHubClient,
		)
		r = r.WithContext(ctx)

		// Route /webhook requests to the webhook handler
		if r.URL.Path == constants.WebhookEndpoint {
			webhookHandler.ServeHTTP(w, r)
			return
		}
		// All other requests go to AgentField handler
		agentFieldHandler.ServeHTTP(w, r)
	})

	s.handler = mainHandler
}

// Start starts the HTTP server
func (s *Server) Start() error {
	port := s.container.EnvConfig.Port

	s.logStartupInfo(port)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: s.handler,
	}

	log.Printf("Webhook endpoint: http://localhost:%s%s", port, constants.WebhookEndpoint)
	return server.ListenAndServe()
}

// logStartupInfo logs server startup information
func (s *Server) logStartupInfo(port string) {
	cfg := s.container.Config
	envCfg := s.container.EnvConfig

	log.Println("Starting GitHub Code Agent...")
	log.Printf("  Node ID: %s", constants.AgentNodeID)
	log.Printf("  Version: %s", constants.AgentVersion)
	log.Printf("  Team ID: %s", constants.AgentTeamID)
	log.Printf("  AgentField URL: %s", envCfg.AgentFieldURL)
	log.Printf("  AI Model: %s", envCfg.AIModel)
	log.Printf("  Mode: %s", cfg.Agent.Mode)
	log.Printf("  Listening on port: %s", port)
}

// Run initializes and starts the application
func Run() error {
	// Bootstrap application
	bootstrap := NewBootstrap()
	container, err := bootstrap.Initialize()
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Create and setup server
	srv := NewServer(container)
	srv.RegisterAgents()
	srv.SetupRoutes()

	// Start server
	return srv.Start()
}
