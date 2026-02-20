package context

import (
	"context"
	"testing"

	"github.com/Agent-Field/agentfield/sdk/go/agent"
	"github.com/yourorg/github-code-agent/pkg/config"
	ghpkg "github.com/yourorg/github-code-agent/pkg/github"
)

// Test context keys - using the existing contextKey type from context.go
const (
	standardKey contextKey = "standard_key"
	baseKey     contextKey = "base_key"
	key1        contextKey = "key1"
	key2        contextKey = "key2"
)

// MockAgent is a simple mock for testing
type MockAgent struct{}

func (m *MockAgent) AI(ctx context.Context, prompt string, opts ...interface{}) (interface{}, error) {
	return nil, nil
}

// MockConfig is a simple mock for testing
type MockConfig struct{}

// MockGitHubClient is a simple mock for testing
type MockGitHubClient struct{}

func TestWithAgent(t *testing.T) {
	ctx := context.Background()
	mockAgent := &agent.Agent{}

	newCtx := WithAgent(ctx, mockAgent)

	if newCtx == ctx {
		t.Error("WithAgent should return a new context")
	}

	retrieved, ok := GetAgent(newCtx)
	if !ok {
		t.Fatal("GetAgent should return true after WithAgent")
	}
	if retrieved != mockAgent {
		t.Error("GetAgent should return the same agent instance")
	}
}

func TestGetAgent(t *testing.T) {
	t.Run("agent present", func(t *testing.T) {
		ctx := context.Background()
		mockAgent := &agent.Agent{}
		ctx = WithAgent(ctx, mockAgent)

		retrieved, ok := GetAgent(ctx)
		if !ok {
			t.Fatal("GetAgent should return true")
		}
		if retrieved != mockAgent {
			t.Error("GetAgent should return the correct agent")
		}
	})

	t.Run("agent not present", func(t *testing.T) {
		ctx := context.Background()

		retrieved, ok := GetAgent(ctx)
		if ok {
			t.Error("GetAgent should return false when agent not present")
		}
		if retrieved != nil {
			t.Error("GetAgent should return nil when agent not present")
		}
	})
}

func TestMustGetAgent(t *testing.T) {
	t.Run("agent present", func(t *testing.T) {
		ctx := context.Background()
		mockAgent := &agent.Agent{}
		ctx = WithAgent(ctx, mockAgent)

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustGetAgent should not panic when agent is present")
			}
		}()

		retrieved := MustGetAgent(ctx)
		if retrieved != mockAgent {
			t.Error("MustGetAgent should return the correct agent")
		}
	})

	t.Run("agent not present", func(t *testing.T) {
		ctx := context.Background()

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetAgent should panic when agent is not present")
			}
		}()

		MustGetAgent(ctx)
	})
}

func TestWithConfig(t *testing.T) {
	ctx := context.Background()
	mockConfig := &config.Config{}

	newCtx := WithConfig(ctx, mockConfig)

	if newCtx == ctx {
		t.Error("WithConfig should return a new context")
	}

	retrieved, ok := GetConfig(newCtx)
	if !ok {
		t.Fatal("GetConfig should return true after WithConfig")
	}
	if retrieved != mockConfig {
		t.Error("GetConfig should return the same config instance")
	}
}

func TestGetConfig(t *testing.T) {
	t.Run("config present", func(t *testing.T) {
		ctx := context.Background()
		mockConfig := &config.Config{}
		ctx = WithConfig(ctx, mockConfig)

		retrieved, ok := GetConfig(ctx)
		if !ok {
			t.Fatal("GetConfig should return true")
		}
		if retrieved != mockConfig {
			t.Error("GetConfig should return the correct config")
		}
	})

	t.Run("config not present", func(t *testing.T) {
		ctx := context.Background()

		retrieved, ok := GetConfig(ctx)
		if ok {
			t.Error("GetConfig should return false when config not present")
		}
		if retrieved != nil {
			t.Error("GetConfig should return nil when config not present")
		}
	})
}

func TestMustGetConfig(t *testing.T) {
	t.Run("config present", func(t *testing.T) {
		ctx := context.Background()
		mockConfig := &config.Config{}
		ctx = WithConfig(ctx, mockConfig)

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustGetConfig should not panic when config is present")
			}
		}()

		retrieved := MustGetConfig(ctx)
		if retrieved != mockConfig {
			t.Error("MustGetConfig should return the correct config")
		}
	})

	t.Run("config not present", func(t *testing.T) {
		ctx := context.Background()

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetConfig should panic when config is not present")
			}
		}()

		MustGetConfig(ctx)
	})
}

func TestWithGitHubClient(t *testing.T) {
	ctx := context.Background()
	mockClient := &ghpkg.Client{}

	newCtx := WithGitHubClient(ctx, mockClient)

	if newCtx == ctx {
		t.Error("WithGitHubClient should return a new context")
	}

	retrieved, ok := GetGitHubClient(newCtx)
	if !ok {
		t.Fatal("GetGitHubClient should return true after WithGitHubClient")
	}
	if retrieved != mockClient {
		t.Error("GetGitHubClient should return the same client instance")
	}
}

func TestGetGitHubClient(t *testing.T) {
	t.Run("client present", func(t *testing.T) {
		ctx := context.Background()
		mockClient := &ghpkg.Client{}
		ctx = WithGitHubClient(ctx, mockClient)

		retrieved, ok := GetGitHubClient(ctx)
		if !ok {
			t.Fatal("GetGitHubClient should return true")
		}
		if retrieved != mockClient {
			t.Error("GetGitHubClient should return the correct client")
		}
	})

	t.Run("client not present", func(t *testing.T) {
		ctx := context.Background()

		retrieved, ok := GetGitHubClient(ctx)
		if ok {
			t.Error("GetGitHubClient should return false when client not present")
		}
		if retrieved != nil {
			t.Error("GetGitHubClient should return nil when client not present")
		}
	})
}

func TestMustGetGitHubClient(t *testing.T) {
	t.Run("client present", func(t *testing.T) {
		ctx := context.Background()
		mockClient := &ghpkg.Client{}
		ctx = WithGitHubClient(ctx, mockClient)

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("MustGetGitHubClient should not panic when client is present")
			}
		}()

		retrieved := MustGetGitHubClient(ctx)
		if retrieved != mockClient {
			t.Error("MustGetGitHubClient should return the correct client")
		}
	})

	t.Run("client not present", func(t *testing.T) {
		ctx := context.Background()

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGetGitHubClient should panic when client is not present")
			}
		}()

		MustGetGitHubClient(ctx)
	})
}

func TestWithAll(t *testing.T) {
	ctx := context.Background()
	mockAgent := &agent.Agent{}
	mockConfig := &config.Config{}
	mockClient := &ghpkg.Client{}

	newCtx := WithAll(ctx, mockAgent, mockConfig, mockClient)

	if newCtx == ctx {
		t.Error("WithAll should return a new context")
	}

	// Verify all values are present
	retrievedAgent, ok := GetAgent(newCtx)
	if !ok {
		t.Error("WithAll should add agent to context")
	}
	if retrievedAgent != mockAgent {
		t.Error("WithAll should add correct agent")
	}

	retrievedConfig, ok := GetConfig(newCtx)
	if !ok {
		t.Error("WithAll should add config to context")
	}
	if retrievedConfig != mockConfig {
		t.Error("WithAll should add correct config")
	}

	retrievedClient, ok := GetGitHubClient(newCtx)
	if !ok {
		t.Error("WithAll should add GitHub client to context")
	}
	if retrievedClient != mockClient {
		t.Error("WithAll should add correct GitHub client")
	}
}

func TestWithAllPartialValues(t *testing.T) {
	ctx := context.Background()
	mockAgent := &agent.Agent{}
	mockConfig := &config.Config{}
	mockClient := &ghpkg.Client{}

	// Add values one at a time
	ctx = WithAgent(ctx, mockAgent)
	ctx = WithConfig(ctx, mockConfig)
	ctx = WithGitHubClient(ctx, mockClient)

	// Verify all values are present
	retrievedAgent, ok := GetAgent(ctx)
	if !ok {
		t.Error("Agent should be present")
	}
	if retrievedAgent != mockAgent {
		t.Error("Agent should be correct")
	}

	retrievedConfig, ok := GetConfig(ctx)
	if !ok {
		t.Error("Config should be present")
	}
	if retrievedConfig != mockConfig {
		t.Error("Config should be correct")
	}

	retrievedClient, ok := GetGitHubClient(ctx)
	if !ok {
		t.Error("GitHub client should be present")
	}
	if retrievedClient != mockClient {
		t.Error("GitHub client should be correct")
	}
}

func TestContextKeyCollision(t *testing.T) {
	// Test that our context keys don't collide with standard context values
	ctx := context.Background()
	ctx = context.WithValue(ctx, standardKey, "standard_value")

	mockAgent := &agent.Agent{}
	ctx = WithAgent(ctx, mockAgent)

	// Verify standard value is still accessible
	standardValue, ok := ctx.Value(standardKey).(string)
	if !ok {
		t.Error("Standard context value should still be accessible")
	}
	if standardValue != "standard_value" {
		t.Error("Standard context value should be correct")
	}

	// Verify our value is also accessible
	retrievedAgent, ok := GetAgent(ctx)
	if !ok {
		t.Error("Agent should be accessible")
	}
	if retrievedAgent != mockAgent {
		t.Error("Agent should be correct")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockAgent := &agent.Agent{}
	ctx = WithAgent(ctx, mockAgent)

	// Cancel the context
	cancel()

	// Verify agent is still accessible even after cancellation
	retrievedAgent, ok := GetAgent(ctx)
	if !ok {
		t.Error("Agent should still be accessible after context cancellation")
	}
	if retrievedAgent != mockAgent {
		t.Error("Agent should still be correct after context cancellation")
	}
}

func TestContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	// Wait for deadline to pass
	<-ctx.Done()

	mockAgent := &agent.Agent{}
	ctx = WithAgent(ctx, mockAgent)

	// Verify agent is still accessible even after deadline
	retrievedAgent, ok := GetAgent(ctx)
	if !ok {
		t.Error("Agent should still be accessible after deadline")
	}
	if retrievedAgent != mockAgent {
		t.Error("Agent should still be correct after deadline")
	}
}

func TestMultipleAgents(t *testing.T) {
	ctx := context.Background()
	agent1 := &agent.Agent{}
	agent2 := &agent.Agent{}

	// Add first agent
	ctx = WithAgent(ctx, agent1)
	retrieved1, ok := GetAgent(ctx)
	if !ok || retrieved1 != agent1 {
		t.Error("First agent should be accessible")
	}

	// Overwrite with second agent
	ctx = WithAgent(ctx, agent2)
	retrieved2, ok := GetAgent(ctx)
	if !ok || retrieved2 != agent2 {
		t.Error("Second agent should overwrite first")
	}
}

func TestMultipleConfigs(t *testing.T) {
	ctx := context.Background()
	config1 := &config.Config{}
	config2 := &config.Config{}

	// Add first config
	ctx = WithConfig(ctx, config1)
	retrieved1, ok := GetConfig(ctx)
	if !ok || retrieved1 != config1 {
		t.Error("First config should be accessible")
	}

	// Overwrite with second config
	ctx = WithConfig(ctx, config2)
	retrieved2, ok := GetConfig(ctx)
	if !ok || retrieved2 != config2 {
		t.Error("Second config should overwrite first")
	}
}

func TestMultipleGitHubClients(t *testing.T) {
	ctx := context.Background()
	client1 := &ghpkg.Client{}
	client2 := &ghpkg.Client{}

	// Add first client
	ctx = WithGitHubClient(ctx, client1)
	retrieved1, ok := GetGitHubClient(ctx)
	if !ok || retrieved1 != client1 {
		t.Error("First client should be accessible")
	}

	// Overwrite with second client
	ctx = WithGitHubClient(ctx, client2)
	retrieved2, ok := GetGitHubClient(ctx)
	if !ok || retrieved2 != client2 {
		t.Error("Second client should overwrite first")
	}
}

func TestContextChain(t *testing.T) {
	// Test that context values are inherited through chain
	baseCtx := context.Background()
	baseCtx = context.WithValue(baseCtx, baseKey, "base_value")

	ctx := WithAll(baseCtx, &agent.Agent{}, &config.Config{}, &ghpkg.Client{})

	// Verify base value is still accessible
	baseValue, ok := ctx.Value(baseKey).(string)
	if !ok || baseValue != "base_value" {
		t.Error("Base context value should be inherited")
	}

	// Verify our values are also accessible
	_, ok = GetAgent(ctx)
	if !ok {
		t.Error("Agent should be accessible in chained context")
	}
}

func TestNilValues(t *testing.T) {
	ctx := context.Background()

	// Test adding nil values
	ctx = WithAgent(ctx, nil)
	retrieved, ok := GetAgent(ctx)
	if !ok {
		t.Error("GetAgent should return true even for nil value")
	}
	if retrieved != nil {
		t.Error("GetAgent should return nil for nil value")
	}

	ctx = WithConfig(ctx, nil)
	retrievedConfig, ok := GetConfig(ctx)
	if !ok {
		t.Error("GetConfig should return true even for nil value")
	}
	if retrievedConfig != nil {
		t.Error("GetConfig should return nil for nil value")
	}

	ctx = WithGitHubClient(ctx, nil)
	retrievedClient, ok := GetGitHubClient(ctx)
	if !ok {
		t.Error("GetGitHubClient should return true even for nil value")
	}
	if retrievedClient != nil {
		t.Error("GetGitHubClient should return nil for nil value")
	}
}

func TestContextTypeSafety(t *testing.T) {
	ctx := context.Background()

	// Add wrong type to our keys (shouldn't happen in normal usage, but test type safety)
	ctx = context.WithValue(ctx, keyAgent, "not an agent")

	// GetAgent should return false for wrong type
	_, ok := GetAgent(ctx)
	if ok {
		t.Error("GetAgent should return false for wrong type")
	}
}

func TestContextKeyUniqueness(t *testing.T) {
	// Verify that our context keys are unique
	if keyAgent == keyConfig {
		t.Error("keyAgent and keyConfig should be different")
	}
	if keyAgent == keyGitHubClient {
		t.Error("keyAgent and keyGitHubClient should be different")
	}
	if keyConfig == keyGitHubClient {
		t.Error("keyConfig and keyGitHubClient should be different")
	}
}

func TestWithAgentDoesNotModifyOriginal(t *testing.T) {
	ctx := context.Background()
	mockAgent := &agent.Agent{}

	newCtx := WithAgent(ctx, mockAgent)

	// Original context should not have the agent
	_, ok := GetAgent(ctx)
	if ok {
		t.Error("Original context should not be modified")
	}

	// New context should have the agent
	_, ok = GetAgent(newCtx)
	if !ok {
		t.Error("New context should have the agent")
	}
}

func TestContextWithValuePreservation(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, key1, "value1")
	ctx = context.WithValue(ctx, key2, "value2")

	mockAgent := &agent.Agent{}
	ctx = WithAgent(ctx, mockAgent)

	// Verify all values are preserved
	if ctx.Value(key1) != "value1" {
		t.Error("key1 should be preserved")
	}
	if ctx.Value(key2) != "value2" {
		t.Error("key2 should be preserved")
	}

	_, ok := GetAgent(ctx)
	if !ok {
		t.Error("Agent should be accessible")
	}
}

func TestContextPropagation(t *testing.T) {
	ctx := context.Background()
	mockAgent := &agent.Agent{}
	ctx = WithAgent(ctx, mockAgent)

	// Simulate passing context through function calls
	callDepth := 0
	var recursiveFunc func(context.Context)
	recursiveFunc = func(c context.Context) {
		callDepth++
		if callDepth < 10 {
			recursiveFunc(c)
		}
	}

	recursiveFunc(ctx)

	// Verify agent is still accessible after propagation
	retrieved, ok := GetAgent(ctx)
	if !ok {
		t.Error("Agent should be accessible after propagation")
	}
	if retrieved != mockAgent {
		t.Error("Agent should be correct after propagation")
	}
}
