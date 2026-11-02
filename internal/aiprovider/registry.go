package aiprovider

import (
	"errors"
	"sort"
	"sync"
	"time"
)

var (
	ErrNotFound         = errors.New("AI provider profile not found")
	ErrRevisionConflict = errors.New("AI provider profile revision conflict")
)

// Registry is a concurrency-safe core registry. Durable controller storage can
// persist returned Profiles without learning plaintext credential material.
type Registry struct {
	mu       sync.RWMutex
	profiles map[string]Profile
	policy   ValidationPolicy
	now      func() time.Time
}

func NewRegistry(policy ValidationPolicy) *Registry {
	return &Registry{profiles: make(map[string]Profile), policy: policy, now: time.Now}
}

// Put creates or updates a profile. expectedRevision must match the current
// revision on update; zero creates a new profile.
func (r *Registry) Put(profile Profile, expectedRevision uint64, actor string) (Profile, error) {
	profile = Normalize(profile)
	if err := Validate(profile, r.policy); err != nil {
		return Profile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, exists := r.profiles[profile.ID]
	if exists {
		if expectedRevision != current.Revision {
			return Profile{}, ErrRevisionConflict
		}
		profile.Revision = current.Revision + 1
		profile.CreatedAt = current.CreatedAt
	} else {
		if expectedRevision != 0 {
			return Profile{}, ErrRevisionConflict
		}
		profile.Revision = 1
		profile.CreatedAt = r.now().UTC()
	}
	profile.UpdatedAt = r.now().UTC()
	profile.UpdatedBy = actor
	r.profiles[profile.ID] = cloneProfile(profile)
	return cloneProfile(profile), nil
}

func (r *Registry) Get(id string) (Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.profiles[id]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return cloneProfile(p), nil
}

func (r *Registry) ListRedacted() []View {
	r.mu.RLock()
	defer r.mu.RUnlock()
	views := make([]View, 0, len(r.profiles))
	for _, p := range r.profiles {
		views = append(views, Redacted(p))
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })
	return views
}

func (r *Registry) Route(fallbackGroup string) []Profile {
	r.mu.RLock()
	profiles := make([]Profile, 0, len(r.profiles))
	for _, p := range r.profiles {
		profiles = append(profiles, cloneProfile(p))
	}
	r.mu.RUnlock()
	return Route(profiles, fallbackGroup)
}

func (r *Registry) Delete(id string, expectedRevision uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[id]
	if !ok {
		return ErrNotFound
	}
	if expectedRevision != p.Revision {
		return ErrRevisionConflict
	}
	delete(r.profiles, id)
	return nil
}

func cloneProfile(p Profile) Profile {
	clone := p
	clone.Headers = make(map[string]string, len(p.Headers))
	for key, value := range p.Headers {
		clone.Headers[key] = value
	}
	return clone
}
