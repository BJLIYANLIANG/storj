// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package accounting

import (
	"github.com/zeebo/errs"
	monkit "gopkg.in/spacemonkeygo/monkit.v2"
)

// Error is a standard error class for this package.
var (
	collectorErr = errs.Class("collector error")
	rollupErr    = errs.Class("collector error")
	mon          = monkit.Package()
)
