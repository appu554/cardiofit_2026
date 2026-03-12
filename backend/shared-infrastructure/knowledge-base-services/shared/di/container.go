// Package di provides dependency injection infrastructure for the Knowledge Base services.
// The DI container enables pluggable data sources and extractors while maintaining
// compile-time safety and explicit dependency graphs.
//
// DESIGN PRINCIPLE: "Build Once, Survive 10 Years"
// The container is frozen infrastructure; providers are added/removed freely.
package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
)

// =============================================================================
// CONTAINER INTERFACE
// =============================================================================

// Container is the main dependency injection container
type Container struct {
	mu          sync.RWMutex
	log         *logrus.Entry
	providers   map[reflect.Type]Provider
	singletons  map[reflect.Type]interface{}
	initialized bool
	closed      bool
}

// Provider is a factory function that creates instances
type Provider interface {
	// Provide creates or returns an instance
	Provide(ctx context.Context, c *Container) (interface{}, error)

	// Lifecycle returns the lifecycle type (singleton, transient, scoped)
	Lifecycle() Lifecycle

	// Type returns the type this provider produces
	Type() reflect.Type

	// Dependencies returns types this provider depends on
	Dependencies() []reflect.Type

	// Name returns a human-readable name for logging
	Name() string
}

// Lifecycle determines how instances are managed
type Lifecycle int

const (
	// Singleton means one instance for the container lifetime
	Singleton Lifecycle = iota

	// Transient means a new instance on each request
	Transient

	// Scoped means one instance per scope (e.g., per request)
	Scoped
)

// =============================================================================
// CONTAINER CREATION
// =============================================================================

// NewContainer creates a new DI container
func NewContainer(log *logrus.Entry) *Container {
	return &Container{
		log:        log.WithField("component", "di-container"),
		providers:  make(map[reflect.Type]Provider),
		singletons: make(map[reflect.Type]interface{}),
	}
}

// =============================================================================
// PROVIDER REGISTRATION
// =============================================================================

// Register adds a provider to the container
func (c *Container) Register(provider Provider) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("container is closed")
	}

	t := provider.Type()
	if _, exists := c.providers[t]; exists {
		c.log.WithField("type", t.String()).Warn("Overwriting existing provider")
	}

	c.providers[t] = provider
	c.log.WithFields(logrus.Fields{
		"type":      t.String(),
		"name":      provider.Name(),
		"lifecycle": provider.Lifecycle(),
	}).Debug("Registered provider")

	return nil
}

// RegisterFunc registers a simple factory function as a provider
func (c *Container) RegisterFunc(name string, t reflect.Type, lifecycle Lifecycle, fn ProviderFunc) error {
	return c.Register(&funcProvider{
		name:      name,
		t:         t,
		lifecycle: lifecycle,
		fn:        fn,
	})
}

// RegisterSingleton registers a pre-created singleton instance
func (c *Container) RegisterSingleton(instance interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return fmt.Errorf("container is closed")
	}

	t := reflect.TypeOf(instance)
	c.singletons[t] = instance
	c.providers[t] = &instanceProvider{instance: instance, t: t}

	c.log.WithField("type", t.String()).Debug("Registered singleton instance")
	return nil
}

// =============================================================================
// DEPENDENCY RESOLUTION
// =============================================================================

// Resolve retrieves or creates an instance of the given type
func (c *Container) Resolve(ctx context.Context, t reflect.Type) (interface{}, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, fmt.Errorf("container is closed")
	}
	c.mu.RUnlock()

	return c.resolveWithPath(ctx, t, make(map[reflect.Type]bool))
}

// resolveWithPath resolves with cycle detection
func (c *Container) resolveWithPath(ctx context.Context, t reflect.Type, path map[reflect.Type]bool) (interface{}, error) {
	// Check for cycles
	if path[t] {
		return nil, fmt.Errorf("circular dependency detected for type: %s", t.String())
	}
	path[t] = true
	defer delete(path, t)

	// Check for singleton instance first
	c.mu.RLock()
	if instance, ok := c.singletons[t]; ok {
		c.mu.RUnlock()
		return instance, nil
	}

	provider, ok := c.providers[t]
	c.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("no provider registered for type: %s", t.String())
	}

	// Resolve dependencies first
	for _, dep := range provider.Dependencies() {
		_, err := c.resolveWithPath(ctx, dep, path)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dependency %s for %s: %w", dep.String(), t.String(), err)
		}
	}

	// Create instance
	instance, err := provider.Provide(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("provider %s failed: %w", provider.Name(), err)
	}

	// Store singleton
	if provider.Lifecycle() == Singleton {
		c.mu.Lock()
		c.singletons[t] = instance
		c.mu.Unlock()
	}

	return instance, nil
}

// MustResolve resolves or panics
func (c *Container) MustResolve(ctx context.Context, t reflect.Type) interface{} {
	instance, err := c.Resolve(ctx, t)
	if err != nil {
		panic(fmt.Sprintf("failed to resolve %s: %v", t.String(), err))
	}
	return instance
}

// =============================================================================
// TYPED RESOLUTION HELPERS
// =============================================================================

// ResolveAs resolves and casts to the expected type
func ResolveAs[T any](ctx context.Context, c *Container) (T, error) {
	var zero T
	t := reflect.TypeOf((*T)(nil)).Elem()
	instance, err := c.Resolve(ctx, t)
	if err != nil {
		return zero, err
	}
	result, ok := instance.(T)
	if !ok {
		return zero, fmt.Errorf("resolved instance is not of type %T", zero)
	}
	return result, nil
}

// MustResolveAs resolves and casts or panics
func MustResolveAs[T any](ctx context.Context, c *Container) T {
	result, err := ResolveAs[T](ctx, c)
	if err != nil {
		panic(err)
	}
	return result
}

// =============================================================================
// CONTAINER LIFECYCLE
// =============================================================================

// Initialize initializes all singleton providers
func (c *Container) Initialize(ctx context.Context) error {
	c.mu.Lock()
	if c.initialized {
		c.mu.Unlock()
		return nil
	}
	c.initialized = true
	c.mu.Unlock()

	c.log.Info("Initializing DI container...")

	// Initialize singletons in dependency order
	for t, provider := range c.providers {
		if provider.Lifecycle() == Singleton {
			_, err := c.Resolve(ctx, t)
			if err != nil {
				return fmt.Errorf("failed to initialize singleton %s: %w", provider.Name(), err)
			}
		}
	}

	c.log.WithField("singletons", len(c.singletons)).Info("DI container initialized")
	return nil
}

// Close closes the container and all closeable singletons
func (c *Container) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	c.log.Info("Closing DI container...")

	var lastErr error
	for t, instance := range c.singletons {
		if closer, ok := instance.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				c.log.WithError(err).WithField("type", t.String()).Warn("Error closing singleton")
				lastErr = err
			}
		}
	}

	c.singletons = make(map[reflect.Type]interface{})
	c.log.Info("DI container closed")
	return lastErr
}

// =============================================================================
// INTROSPECTION
// =============================================================================

// Has checks if a type is registered
func (c *Container) Has(t reflect.Type) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.providers[t]
	return ok
}

// ListProviders returns information about all registered providers
func (c *Container) ListProviders() []ProviderInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	infos := make([]ProviderInfo, 0, len(c.providers))
	for t, p := range c.providers {
		_, isSingleton := c.singletons[t]
		infos = append(infos, ProviderInfo{
			Name:         p.Name(),
			Type:         t.String(),
			Lifecycle:    p.Lifecycle(),
			Dependencies: dependencyNames(p.Dependencies()),
			Initialized:  isSingleton,
		})
	}
	return infos
}

// ProviderInfo contains information about a registered provider
type ProviderInfo struct {
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Lifecycle    Lifecycle `json:"lifecycle"`
	Dependencies []string  `json:"dependencies"`
	Initialized  bool      `json:"initialized"`
}

func dependencyNames(deps []reflect.Type) []string {
	names := make([]string, len(deps))
	for i, d := range deps {
		names[i] = d.String()
	}
	return names
}

// =============================================================================
// BUILT-IN PROVIDERS
// =============================================================================

// ProviderFunc is a function that provides an instance
type ProviderFunc func(ctx context.Context, c *Container) (interface{}, error)

// funcProvider wraps a function as a provider
type funcProvider struct {
	name      string
	t         reflect.Type
	lifecycle Lifecycle
	fn        ProviderFunc
	deps      []reflect.Type
}

func (p *funcProvider) Provide(ctx context.Context, c *Container) (interface{}, error) {
	return p.fn(ctx, c)
}

func (p *funcProvider) Lifecycle() Lifecycle    { return p.lifecycle }
func (p *funcProvider) Type() reflect.Type      { return p.t }
func (p *funcProvider) Dependencies() []reflect.Type { return p.deps }
func (p *funcProvider) Name() string            { return p.name }

// instanceProvider wraps a pre-created instance
type instanceProvider struct {
	instance interface{}
	t        reflect.Type
}

func (p *instanceProvider) Provide(ctx context.Context, c *Container) (interface{}, error) {
	return p.instance, nil
}

func (p *instanceProvider) Lifecycle() Lifecycle    { return Singleton }
func (p *instanceProvider) Type() reflect.Type      { return p.t }
func (p *instanceProvider) Dependencies() []reflect.Type { return nil }
func (p *instanceProvider) Name() string            { return fmt.Sprintf("instance<%s>", p.t.String()) }

// =============================================================================
// SCOPED CONTAINER
// =============================================================================

// Scope creates a child container for scoped resolution
type Scope struct {
	parent   *Container
	mu       sync.RWMutex
	scoped   map[reflect.Type]interface{}
}

// NewScope creates a new scope from a container
func (c *Container) NewScope() *Scope {
	return &Scope{
		parent: c,
		scoped: make(map[reflect.Type]interface{}),
	}
}

// Resolve resolves within this scope
func (s *Scope) Resolve(ctx context.Context, t reflect.Type) (interface{}, error) {
	// Check scoped instances first
	s.mu.RLock()
	if instance, ok := s.scoped[t]; ok {
		s.mu.RUnlock()
		return instance, nil
	}
	s.mu.RUnlock()

	// Check if provider exists and is scoped
	s.parent.mu.RLock()
	provider, ok := s.parent.providers[t]
	s.parent.mu.RUnlock()

	if ok && provider.Lifecycle() == Scoped {
		// Create scoped instance
		instance, err := provider.Provide(ctx, s.parent)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.scoped[t] = instance
		s.mu.Unlock()
		return instance, nil
	}

	// Delegate to parent for singletons and transients
	return s.parent.Resolve(ctx, t)
}

// Close closes all scoped instances
func (s *Scope) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lastErr error
	for _, instance := range s.scoped {
		if closer, ok := instance.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				lastErr = err
			}
		}
	}
	s.scoped = make(map[reflect.Type]interface{})
	return lastErr
}

// =============================================================================
// MODULE SYSTEM
// =============================================================================

// Module groups related providers for organized registration
type Module interface {
	// Name returns the module name
	Name() string

	// Register registers all providers in this module
	Register(c *Container) error

	// Dependencies returns other modules this module depends on
	Dependencies() []Module
}

// RegisterModules registers multiple modules in dependency order
func (c *Container) RegisterModules(modules ...Module) error {
	registered := make(map[string]bool)
	return c.registerModulesRecursive(modules, registered)
}

func (c *Container) registerModulesRecursive(modules []Module, registered map[string]bool) error {
	for _, mod := range modules {
		if registered[mod.Name()] {
			continue
		}

		// Register dependencies first
		if err := c.registerModulesRecursive(mod.Dependencies(), registered); err != nil {
			return err
		}

		// Register this module
		if err := mod.Register(c); err != nil {
			return fmt.Errorf("failed to register module %s: %w", mod.Name(), err)
		}
		registered[mod.Name()] = true
		c.log.WithField("module", mod.Name()).Info("Registered module")
	}
	return nil
}

// =============================================================================
// TYPE UTILITIES
// =============================================================================

// TypeOf returns the reflect.Type for a type parameter
func TypeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// InterfaceType returns the reflect.Type for an interface
func InterfaceType[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}
