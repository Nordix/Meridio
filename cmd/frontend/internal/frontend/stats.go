/*
Copyright (c) 2023 Nordix Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package frontend

import (
	"context"
	"strings"

	"github.com/nordix/meridio/cmd/frontend/internal/bird"
	"github.com/nordix/meridio/pkg/log"
)

const routePrintRateLimit int = 60 // max iterations (calls to limit) to wait before printing minor changes
const routeThreshold uint64 = 1000 // diff threshold below which rate limiter is active

type RateLimiter struct {
	limit     int    // number of iterations to wait before lifting the rate limiter imposed block assuming diff is below threshold
	threshold uint64 // diff threshold below which RateLimiter activates/is active
}

func NewRateLimiter(limit int, threshold uint64) *RateLimiter {
	return &RateLimiter{
		limit:     limit,
		threshold: threshold,
	}
}

// Active -
// Returns true if rate limiter is active
func (rl *RateLimiter) Active() bool {
	return rl.limit > 0
}

// Permit -
// Returns false if rate limiter says NOT to permit actions, true otherwise.
func (rl *RateLimiter) Permit(diff uint64) bool {
	if diff < rl.threshold {
		// no diff, disable rate limiter, but still no permit (new value equals the last printed value)
		if diff == 0 {
			rl.limit = 0
			return false
		}
		// minor change, enable rate limiter if not active and no permit
		if !rl.Active() {
			rl.limit = routePrintRateLimit
			return false
		}
		// rate limiter already enabled, no permit until limit drops to zero
		rl.limit = rl.limit - 1
		if rl.limit > 0 {
			return false
		}
	}
	// permit, disable rate limiter if active
	rl.limit = 0
	return true
}

type RouteStats struct {
	lastCount   uint64       // last recorded and printed route count
	rateLimiter *RateLimiter // to avoid spamming logs on frequent changes
}

func NewRouteStats() *RouteStats {
	return &RouteStats{
		rateLimiter: NewRateLimiter(routePrintRateLimit, routeThreshold),
	}
}

// LimiterActive -
// Returns true if a route limiter is set and is actived (i.e. blocks actions)
func (rs *RouteStats) LimiterActive() bool {
	return rs.rateLimiter != nil && rs.rateLimiter.Active()
}

// Skip -
// Returns true if a rate limiter is set and its Permit() func returns false
func (rs *RouteStats) Skip(diff uint64) bool {
	if rs.rateLimiter == nil {
		return false
	}
	return !rs.rateLimiter.Permit(diff)
}

// checkRoutes -
// Checks and logs number of routes in routing suite.
// Employes some basic rate limiter to reduce spamming in case of minor changes.
//
// Note: IMHO it's not the best concept; BIRD routing suite seems to block the
// CLI operation while the routing entries are being processed or a reconfiguration
// is ongoing.. In my tests transfering 100k routes takes 2-3 seconds. The checker
// is not able to print any information before that.
// Consider there are even more routes and OMM just kills the container due to its
// memory usage, but there's nothing in the logs...
func (fes *FrontEndService) checkRoutes(ctx context.Context, stats *RouteStats, lp string) {
	logger := log.FromContextOrGlobal(ctx)
	routeOut, err := fes.routingService.ShowRouteCount(ctx, lp)
	if err != nil {
		logger.Info("Route check failed", "err", err, "out", strings.Split(routeOut, "\n"))
		return
	}
	if routeCount, err := bird.ParseRouteCount(routeOut); err == nil && stats.lastCount != routeCount || stats.LimiterActive() {
		// XXX: Number of routes might fluctuate, use basic log rate limiter:
		// Log right away if new number differs from previous value by more than 1000.
		// Otherwise start rate limiter, i.e. wait for either the change to exceed the
		// threshold or for pre-defined number of checks to print the most recent value.
		var diff uint64
		lastCount := stats.lastCount
		if routeCount < lastCount {
			diff = lastCount - routeCount
		} else {
			diff = routeCount - lastCount
		}
		if stats.Skip(diff) {
			return
		}
		// The number of routes maintained by routing service (includes all routes)
		logger.Info("Total number routes", "count", routeCount, "out", strings.Split(routeOut, "\n"))
		stats.lastCount = routeCount
		// Fetch memory usage information
		if memOut, err := fes.routingService.ShowMemory(ctx, lp); err == nil {
			logger.Info("Memory usage", "memor", strings.Split(memOut, "\n"))
		}
	}
}
