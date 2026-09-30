//go:build windows

package localsandbox

import (
	"github.com/acrbaran/rag/internal/localsandbox/core"
	"github.com/acrbaran/rag/internal/localsandbox/winhost"
)

func NewBackend() (core.Backend, error) { return winhost.New() }
