package onboarding

import (
	"context"
	"errors"
	"fmt"

	"github.com/kyle/scout/open-core/apps/api/internal/configfile"
	"github.com/kyle/scout/open-core/apps/api/internal/settings"
)

// Status is the snapshot returned by Service.GetStatus.
type Status struct {
	// Enabled reports whether the onboarding flow is active (i.e. not disabled).
	Enabled bool

	// Complete reports whether the onboarding.complete key is set in the
	// settings repository.
	Complete bool

	// EnvFilePath is the effective env-file path from Config, empty in native
	// mode.
	EnvFilePath string
}

// Service encapsulates all business logic for the onboarding flow.
type Service struct {
	cfg  Config
	repo *settings.Repository
}

// NewService constructs a Service. It returns an error if the Config is
// inconsistent (reserved for future validation; currently always nil).
func NewService(cfg Config, repo *settings.Repository) (*Service, error) {
	if repo == nil {
		return nil, errors.New("onboarding: settings repository must not be nil")
	}
	return &Service{cfg: cfg, repo: repo}, nil
}

// IsComplete reports whether the onboarding completion flag has been stored in
// the settings repository.
func (s *Service) IsComplete(ctx context.Context) (bool, error) {
	val, found, err := s.repo.Get(ctx, s.cfg.CompletionKey)
	if err != nil {
		return false, fmt.Errorf("onboarding: IsComplete: %w", err)
	}
	return found && val == "true", nil
}

// GetStatus returns a point-in-time snapshot of the onboarding state.
func (s *Service) GetStatus(ctx context.Context) (Status, error) {
	complete, err := s.IsComplete(ctx)
	if err != nil {
		return Status{}, err
	}
	return Status{
		Enabled:     !s.cfg.Disabled,
		Complete:    complete,
		EnvFilePath: s.cfg.EnvFilePath,
	}, nil
}

// Save writes the supplied key/value pairs to the env file (Docker mode only)
// and marks onboarding as complete. It returns an error when:
//   - onboarding is disabled
//   - in Docker mode: values is nil or empty, or any value is an empty string
func (s *Service) Save(ctx context.Context, values map[string]string) error {
	if s.cfg.Disabled {
		return errors.New("onboarding: save rejected — onboarding is disabled")
	}

	if s.cfg.DockerMode {
		if len(values) == 0 {
			return errors.New("onboarding: save rejected — no values provided in Docker mode")
		}
		for k, v := range values {
			if v == "" {
				return fmt.Errorf("onboarding: save rejected — empty value for key %q in Docker mode", k)
			}
		}
		if _, err := configfile.UpdateFile(s.cfg.EnvFilePath, values, nil); err != nil {
			return fmt.Errorf("onboarding: writing env file: %w", err)
		}
	}

	if err := s.repo.Set(ctx, s.cfg.CompletionKey, "true"); err != nil {
		return fmt.Errorf("onboarding: marking complete: %w", err)
	}
	return nil
}

// Reset clears the completion flag. It is idempotent — resetting an already-
// reset service is not an error.
func (s *Service) Reset(ctx context.Context) error {
	if err := s.repo.Delete(ctx, s.cfg.CompletionKey); err != nil {
		return fmt.Errorf("onboarding: reset: %w", err)
	}
	return nil
}
