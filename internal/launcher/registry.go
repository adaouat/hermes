package launcher

import "github.com/adaouat/hermes/internal/env"

// Registry holds every available Launcher and selects one at runtime. def is
// always returned by Default and is never included in Detect's
// auto-detection loop (the generic adapter has no reliable env signal — it's
// the deliberate fallback, not something to race against real signals).
type Registry struct {
	def    Launcher
	others []Launcher
	byName map[string]Launcher
}

// NewRegistry builds a Registry. def is the launcher Default returns (the
// generic adapter in practice); others are probed in order by Detect.
func NewRegistry(def Launcher, others ...Launcher) *Registry {
	r := &Registry{def: def, byName: map[string]Launcher{def.Name(): def}}
	for _, l := range others {
		r.Register(l)
	}
	return r
}

// Register adds l to the set Get and Detect can return.
func (r *Registry) Register(l Launcher) {
	r.others = append(r.others, l)
	r.byName[l.Name()] = l
}

// Get returns the launcher registered under name.
func (r *Registry) Get(name string) (Launcher, bool) {
	l, ok := r.byName[name]
	return l, ok
}

// Detect returns the first registered (non-default) launcher whose
// Detect(e) is true, in registration order. Returns false if none match —
// callers fall back to Default() (ADR-0002 O1: "no launcher resolved is not
// an error").
func (r *Registry) Detect(e env.Env) (Launcher, bool) {
	for _, l := range r.others {
		if l.Detect(e) {
			return l, true
		}
	}
	return nil, false
}

// Default returns the launcher used when no flag, env var, or Detect signal
// resolves one (the generic adapter, per ADR-0002 O3).
func (r *Registry) Default() Launcher {
	return r.def
}
