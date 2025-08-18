package server

import (
	"context"

	"github.com/kyverno/kyverno-envoy-plugin/pkg/log"
)

var serverLogger = log.RegisterScope("server", "Authorization Server")

type Server interface {
	Run(context.Context) error
}

type ServerFunc func(context.Context) error

func (f ServerFunc) Run(ctx context.Context) error {
	return f(ctx)
}
