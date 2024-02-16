package credentials

import (
	"context"
	"sync"

	"github.com/nordix/meridio/pkg/log"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/credentials"
)

var (
	mu         sync.Mutex
	x509Source *workloadapi.X509Source
)

func GetClient(ctx context.Context) credentials.TransportCredentials {
	source := GetX509Source(ctx)
	if source == nil {
		return nil
	}
	tlsAuthorizer := tlsconfig.AuthorizeAny()
	return grpccredentials.MTLSClientCredentials(source, source, tlsAuthorizer)
}

func GetServer(ctx context.Context) credentials.TransportCredentials {
	source := GetX509Source(ctx)
	if source == nil {
		return nil
	}
	tlsAuthorizer := tlsconfig.AuthorizeAny()
	return grpccredentials.MTLSServerCredentials(source, source, tlsAuthorizer)
}

func GetServerWithSource(ctx context.Context, source *workloadapi.X509Source) credentials.TransportCredentials {
	tlsAuthorizer := tlsconfig.AuthorizeAny()
	return grpccredentials.MTLSServerCredentials(source, source, tlsAuthorizer)
}

// GetX509Source -
// Returns a X509 source. Creates a new source if none exists yet.
//
// Note: Avoid creating new X509 whenever invoked as it each source
// consumes additional resources.
// TODO: According to the description of NewX509Source() the source
// should be closed to release underlying resources. So, consider
// adding a function that would take care of closing the source.
func GetX509Source(ctx context.Context) *workloadapi.X509Source {
	mu.Lock()
	defer mu.Unlock()

	if x509Source == nil {
		logger := log.FromContextOrGlobal(ctx)
		source, err := workloadapi.NewX509Source(context.Background())
		if err != nil {
			logger.Error(err, "error getting x509 source")
			return nil
		}
		if svid, err := source.GetX509SVID(); err != nil {
			logger.Error(err, "error getting x509 svid")
		} else {
			logger.Info("GetX509Source", "sVID", svid.ID)
		}
		x509Source = source
	}
	return x509Source
}
