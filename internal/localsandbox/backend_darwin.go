//go:build darwin

package localsandbox

import (
	"context"

	"github.com/acrbaran/rag/internal/localsandbox/core"
	"github.com/acrbaran/rag/internal/localsandbox/seatbelt"
	"github.com/acrbaran/rag/internal/logger"
)

// NewBackend returns the platform backend. The build tag on this file is the
// only place the OS is selected; no other code may branch on runtime.GOOS.
func NewBackend() (core.Backend, error) {
	logger.Infof(context.Background(), "[LocalSandbox] using seatbelt backend")
	return seatbelt.New()
}
