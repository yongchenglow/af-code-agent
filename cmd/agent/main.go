package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/Agent-Field/agentfield/sdk/go/ai"
	"github.com/joho/godotenv"

	"github.com/yourorg/github-code-agent/features/analyzer"
	"github.com/yourorg/github-code-agent/features/fixer"
	"github.com/yourorg/github-code-agent/features/gitops"
	"github.com/yourorg/github-code-agent/features/reviewer"
	"github.com/yourorg/github-code-agent/features/standards"
	"github.com/yourorg/github-code-agent/features/webhook"
	"github.com/yourorg/github-code-agent/pkg/config"
	"github.com/yourorg/github-code-agent/pkg/github"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load application configuration
	cfg, err := config.LoadConfig(".github/code-agent.yml")
	if err != nil {
		log.Printf("Warning: Failed to load config: %v", err)
		log.Println("Using default configuration")
		cfg = config.DefaultConfig()
	}

	// Load environment configuration
	envCfg, err := config.LoadEnvironmentConfig()
	if err != nil {
		log.Fatalf("Failed to load environment config: %v", err)
	}

	// Configure AI (supports OpenAI or OpenRouter)
	// For DeepSeek via OpenRouter:
	// export OPENROUTER_API_KEY="sk-or-v1-..."
	// export AI_MODEL="deepseek/deepseek-chat"
	aiConfig := ai.DefaultConfig() // Reads from environment

	// Create AgentField agent
	app, err := agent.New(agent.Config{
		NodeID:        "github-code-agent",
		Version:       "1.0.0",
		TeamID:        "code-review",
		AgentFieldURL: envCfg.AgentFieldURL,
		AIConfig:      aiConfig,
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create GitHub client wrapper
	ghClientWrapper, err := github.NewClient(envCfg.GitHubToken)
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}

	// Get the basic GitHub client for gitops
	ghClient := ghClientWrapper.GetClient()

	// Store config in context for reasoners to access
	ctx := context.WithValue(context.Background(), "config", cfg)
	ctx = context.WithValue(ctx, "agent", app)

	// Register all feature reasoners
	log.Println("Registering reasoners...")
	webhook.RegisterReasoners(app)
	analyzer.RegisterReasoners(app)
	reviewer.RegisterReasoners(app)         // Phase 2 ✓
	standards.RegisterReasoners(app, cfg)   // Phase 2 ✓
	fixer.RegisterReasoners(app)            // Phase 3 ✓
	gitops.RegisterReasoners(app, ghClient) // Phase 3 ✓

	// Create custom HTTP server with webhook handler
	mux := http.NewServeMux()

	// Register the custom webhook handler (validates GitHub signatures)
	webhook.RegisterWebhookHandler(mux, app, envCfg.GitHubWebhookSecret)

	// Register the AgentField handler for all other routes
	mux.Handle("/", app.Handler())

	// Start HTTP server
	port := envCfg.Port
	log.Println("Starting GitHub Code Agent...")
	log.Printf("  Node ID: %s", "github-code-agent")
	log.Printf("  Version: %s", "1.0.0")
	log.Printf("  Team ID: %s", "code-review")
	log.Printf("  AgentField URL: %s", getEnv("AGENTFIELD_URL", "http://localhost:8080"))
	log.Printf("  AI Model: %s", aiConfig.Model)
	log.Printf("  Mode: %s", cfg.Agent.Mode)
	log.Printf("  Listening on port: %s", port)

	// Start server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Webhook endpoint: http://localhost:%s/webhook", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// getEnv retrieves an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
