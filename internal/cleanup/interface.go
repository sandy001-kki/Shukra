package cleanup

import (
	"context"
	"errors"
)

type Hook interface {
	Name() string
	Cleanup(ctx context.Context, env CleanupTarget) error
}

type CleanupTarget struct {
	Name              string
	Namespace         string
	DatabaseMode      string
	DatabaseSecret    string
	BackupDestination string
	IngressHost       string
}

var ErrTransient = errors.New("transient cleanup error")
var ErrPermanent = errors.New("permanent cleanup error")
