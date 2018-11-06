// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package accounting

import (
	"context"
	"time"

	"go.uber.org/zap"

	"storj.io/storj/pkg/overlay"
	"storj.io/storj/pkg/pointerdb"
	"storj.io/storj/pkg/provider"
)

// Config contains configurable values accounting
type Config struct {
	Interval time.Duration `help:"how frequently checker should audit segments" default:"30s"`
}

// Initialize a Accounting Collector struct
func (c Config) initialize(ctx context.Context) (Collector, error) {
	pointerdb := pointerdb.LoadFromContext(ctx)
	overlay := overlay.LoadServerFromContext(ctx)
	return NewCollector(pointerdb, overlay, 0, zap.L(), c.Interval), nil
}

// Run runs the checker with configured values
func (c Config) Run(ctx context.Context, server *provider.Provider) (err error) {
	collect, err := c.initialize(ctx)
	if err != nil {
		return err
	}

	// TODO(coyle): we need to figure out how to propagate the error up to cancel the service
	go func() {
		if err := collect.Run(ctx); err != nil {
			zap.L().Error("Error running collector", zap.Error(err))
		}
	}()

	return server.Run(ctx)
}
