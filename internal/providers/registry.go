package providers

import (
	"fmt"
	"sync"
)

// Registry manages available generative AI providers (images and videos).
type Registry struct {
	mu             sync.RWMutex
	providers      map[string]Provider
	videoProviders map[string]VideoProvider
}

// NewRegistry initializes an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers:      make(map[string]Provider),
		videoProviders: make(map[string]VideoProvider),
	}
}

// Register adds or replaces a provider in the registry. If the provider implements
// VideoProvider, it is automatically discovered and indexed.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
	if vp, ok := p.(VideoProvider); ok {
		r.videoProviders[vp.Name()] = vp
	}
}

// RegisterVideo explicitly registers a standalone VideoProvider.
func (r *Registry) RegisterVideo(vp VideoProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.videoProviders[vp.Name()] = vp
}

// Get retrieves an image provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found in registry", name)
	}
	return p, nil
}

// GetVideo retrieves a video provider by name.
func (r *Registry) GetVideo(name string) (VideoProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	vp, ok := r.videoProviders[name]
	if !ok {
		return nil, fmt.Errorf("video provider %q not found in registry", name)
	}
	return vp, nil
}

// List returns names of all registered image providers.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// ListVideo returns names of all registered video providers.
func (r *Registry) ListVideo() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.videoProviders))
	for name := range r.videoProviders {
		names = append(names, name)
	}
	return names
}
