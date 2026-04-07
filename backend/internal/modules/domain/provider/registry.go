package provider

import "strings"

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry(cloudflareBaseURL string, spaceshipBaseURL string) *Registry {
	return &Registry{
		adapters: map[string]Adapter{
			"cloudflare": NewCloudflareAdapter(cloudflareBaseURL, nil),
			"spaceship":  NewSpaceshipAdapter(spaceshipBaseURL, nil),
		},
	}
}

func (r *Registry) Get(providerName string) (Adapter, bool) {
	if r == nil {
		return nil, false
	}
	adapter, ok := r.adapters[strings.TrimSpace(strings.ToLower(providerName))]
	return adapter, ok
}
