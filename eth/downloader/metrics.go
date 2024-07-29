// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/X7-Group/x7chain/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("x7c/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("x7c/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("x7c/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("x7c/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("x7c/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("x7c/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("x7c/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("x7c/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("x7c/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("x7c/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("x7c/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("x7c/downloader/receipts/timeout", nil)

	throttleCounter = metrics.NewRegisteredCounter("x7c/downloader/throttle", nil)
)
