package fcmclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Factory resolves FCM messaging clients from wire JSON or env bootstrap.
type Factory struct {
	envClient *messaging.Client
	mu        sync.Mutex
	cache     map[string]*messaging.Client
}

// NewFactory constructs a resolver. envClient may be nil when GOOGLE_APPLICATION_CREDENTIALS is unset.
func NewFactory(envClient *messaging.Client) *Factory {
	return &Factory{envClient: envClient, cache: make(map[string]*messaging.Client)}
}

// Resolve returns a messaging client for SendPush. wireJSON takes precedence over env bootstrap.
func (f *Factory) Resolve(ctx context.Context, wireJSON string) (*messaging.Client, error) {
	trimmed := strings.TrimSpace(wireJSON)
	if trimmed != "" {
		if _, err := ValidateServiceAccountJSON(trimmed); err != nil {
			return nil, err
		}
		key := cacheKey(trimmed)
		f.mu.Lock()
		defer f.mu.Unlock()
		if c, ok := f.cache[key]; ok {
			return c, nil
		}
		app, err := firebase.NewApp(ctx, nil, option.WithCredentialsJSON([]byte(trimmed)))
		if err != nil {
			return nil, fmt.Errorf("firebase app init failed: %w", err)
		}
		client, err := app.Messaging(ctx)
		if err != nil {
			return nil, fmt.Errorf("firebase messaging init failed: %w", err)
		}
		f.cache[key] = client
		return client, nil
	}
	if f.envClient != nil {
		return f.envClient, nil
	}
	return nil, fmt.Errorf("FCM is not configured")
}

// Probe validates credentials and initializes Firebase messaging without sending a notification.
func (f *Factory) Probe(ctx context.Context, wireJSON string) (projectID string, err error) {
	projectID, err = ValidateServiceAccountJSON(wireJSON)
	if err != nil {
		return "", err
	}
	_, err = f.Resolve(ctx, wireJSON)
	if err != nil {
		return projectID, err
	}
	return projectID, nil
}

func cacheKey(json string) string {
	sum := sha256.Sum256([]byte(json))
	return hex.EncodeToString(sum[:])
}
