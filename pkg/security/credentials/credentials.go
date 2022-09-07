package credentials

import (
	"context"

	"github.com/nordix/meridio/pkg/log"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/credentials"
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

func GetX509Source(ctx context.Context) *workloadapi.X509Source {
	// todo: retry if source in nil or empty
	logger := log.FromContextOrGlobal(ctx)
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		logger.Error(err, "error getting x509 source")
		return nil
	}
	if svid, err := source.GetX509SVID(); err != nil {
		logger.Error(err, "error getting x509 svid")
	} else {
		logger.Info("GetX509Source", "sVID", svid.ID)
	}
	return source
}
