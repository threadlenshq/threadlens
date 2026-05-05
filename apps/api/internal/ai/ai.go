// Package ai provides AI provider management and generation services.
// New() is an alias for NewService(nil) and exists for callers that do not
// yet have a repository reference (e.g. QueryService stub).
package ai

// New returns a new AI Service instance with no repository and the default
// production providers.  Callers that need task-model resolution should use
// NewService(repo) instead.
func New() *Service {
	return NewService(nil)
}
