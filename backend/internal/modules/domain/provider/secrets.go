package provider

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

var ErrSecretRefNotFound = errors.New("secret ref not found")
var ErrInvalidProviderSecret = errors.New("invalid provider secret")

func ResolveSecretRef(secretRef string) (string, error) {
	trimmed := strings.TrimSpace(secretRef)
	if trimmed == "" {
		return "", ErrInvalidProviderSecret
	}
	if !strings.HasPrefix(trimmed, "env:") {
		return trimmed, nil
	}

	envKey := strings.TrimSpace(strings.TrimPrefix(trimmed, "env:"))
	if envKey == "" {
		return "", ErrInvalidProviderSecret
	}

	value, ok := os.LookupEnv(envKey)
	if !ok || strings.TrimSpace(value) == "" {
		return "", ErrSecretRefNotFound
	}
	return strings.TrimSpace(value), nil
}

type cloudflareSecret struct {
	APIToken  string `json:"apiToken"`
	Token     string `json:"token"`
	APIKey    string `json:"apiKey"`
	Email     string `json:"email"`
	AuthEmail string `json:"authEmail"`
}

type CloudflareCredentials struct {
	APIToken string
	APIKey   string
	Email    string
}

func ResolveCloudflareCredentials(secretRef string) (CloudflareCredentials, error) {
	resolved, err := ResolveSecretRef(secretRef)
	if err != nil {
		return CloudflareCredentials{}, err
	}

	var payload cloudflareSecret
	if json.Unmarshal([]byte(resolved), &payload) == nil {
		credentials := CloudflareCredentials{
			APIToken: strings.TrimSpace(payload.APIToken),
			APIKey:   strings.TrimSpace(payload.APIKey),
			Email:    strings.TrimSpace(payload.Email),
		}
		if credentials.APIToken == "" {
			credentials.APIToken = strings.TrimSpace(payload.Token)
		}
		if credentials.Email == "" {
			credentials.Email = strings.TrimSpace(payload.AuthEmail)
		}

		switch {
		case credentials.APIToken != "":
			return credentials, nil
		case credentials.APIKey != "" && credentials.Email != "":
			return credentials, nil
		}
	}

	if strings.TrimSpace(resolved) == "" {
		return CloudflareCredentials{}, ErrInvalidProviderSecret
	}
	return CloudflareCredentials{
		APIToken: strings.TrimSpace(resolved),
	}, nil
}

func ResolveCloudflareToken(secretRef string) (string, error) {
	credentials, err := ResolveCloudflareCredentials(secretRef)
	if err != nil {
		return "", err
	}
	if credentials.APIToken == "" {
		return "", ErrInvalidProviderSecret
	}
	return credentials.APIToken, nil
}

type SpaceshipCredentials struct {
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
}

func ResolveSpaceshipCredentials(secretRef string) (SpaceshipCredentials, error) {
	resolved, err := ResolveSecretRef(secretRef)
	if err != nil {
		return SpaceshipCredentials{}, err
	}

	var payload SpaceshipCredentials
	if err := json.Unmarshal([]byte(resolved), &payload); err != nil {
		return SpaceshipCredentials{}, ErrInvalidProviderSecret
	}
	if strings.TrimSpace(payload.APIKey) == "" || strings.TrimSpace(payload.APISecret) == "" {
		return SpaceshipCredentials{}, ErrInvalidProviderSecret
	}
	payload.APIKey = strings.TrimSpace(payload.APIKey)
	payload.APISecret = strings.TrimSpace(payload.APISecret)
	return payload, nil
}
