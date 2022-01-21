//
package refresh

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// store sets the context.CancelFunc stored in per Connection.Id metadata.
func store(ctx context.Context, isClient bool, cancel context.CancelFunc) {
	metadata.Map(ctx, isClient).Store(key{}, cancel)
}

// loadAndDelete deletes the context.CancelFunc stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDelete(ctx context.Context, isClient bool) (value context.CancelFunc, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(context.CancelFunc)
	return value, ok
}
