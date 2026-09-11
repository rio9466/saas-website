package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// Application error code for an unavailable readiness dependency.
const CodeDependencyUnavailable = 50001

const msgDependencyUnavailable = "dependency unavailable"

// ReadyChecker is the minimal readiness dependency consumed by the handler.
type ReadyChecker interface {
	Ping(ctx context.Context) error
}

// NamedReadyCheck is one dependency checked by MultiReady.
type NamedReadyCheck struct {
	Name    string
	Checker ReadyChecker
	Timeout time.Duration
}

type multiReady struct {
	checks []NamedReadyCheck
	logger *slog.Logger
}

// MultiReady returns a composite readiness checker. HTTP responses remain
// generic; safe internal logs may name the failed dependency without secrets.
func MultiReady(logger *slog.Logger, checks ...NamedReadyCheck) ReadyChecker {
	return &multiReady{checks: checks, logger: logger}
}

func (m *multiReady) Ping(ctx context.Context) error {
	if m == nil {
		return errors.New(msgDependencyUnavailable)
	}
	for _, check := range m.checks {
		if check.Checker == nil {
			m.logUnavailable(check.Name)
			return errors.New(msgDependencyUnavailable)
		}
		pingCtx := ctx
		cancel := func() {}
		if check.Timeout > 0 {
			pingCtx, cancel = context.WithTimeout(ctx, check.Timeout)
		}
		err := check.Checker.Ping(pingCtx)
		cancel()
		if err != nil {
			m.logUnavailable(check.Name)
			return errors.New(msgDependencyUnavailable)
		}
	}
	return nil
}

func (m *multiReady) logUnavailable(name string) {
	if m.logger == nil {
		return
	}
	if name == "" {
		name = "dependency"
	}
	m.logger.Error("readiness dependency unavailable", "dependency", name)
}

// Healthz returns process liveness without calling dependencies.
func Healthz(c *gin.Context) {
	response.OK(c, map[string]string{"status": "ok"})
}

// Readyz returns readiness based on the injected dependency check.
func Readyz(checker ReadyChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checker == nil {
			response.Error(c, http.StatusServiceUnavailable, CodeDependencyUnavailable, msgDependencyUnavailable)
			return
		}

		if err := checker.Ping(c.Request.Context()); err != nil {
			response.Error(c, http.StatusServiceUnavailable, CodeDependencyUnavailable, msgDependencyUnavailable)
			return
		}

		response.OK(c, map[string]string{"status": "ready"})
	}
}
