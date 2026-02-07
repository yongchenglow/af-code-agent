package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ValidateSignature validates GitHub webhook signature using HMAC-SHA256
func ValidateSignature(payload []byte, signature, secret string) error {
	// GitHub sends signature as "sha256=<hash>"
	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("invalid signature format")
	}

	// Extract the hash part
	receivedHash := strings.TrimPrefix(signature, "sha256=")

	// Compute expected hash
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	// Compare hashes
	if !hmac.Equal([]byte(receivedHash), []byte(expectedHash)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ShouldProcessEvent determines if the event should be processed
func ShouldProcessEvent(eventType, action string, enabledTriggers []string) bool {
	trigger := eventType
	if action != "" {
		trigger = eventType + "." + action
	}

	for _, enabled := range enabledTriggers {
		if enabled == trigger {
			return true
		}
	}

	return false
}
