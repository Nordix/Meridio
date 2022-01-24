package credentials

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffegrpc/grpccredentials"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/credentials"
)

func GetClient(ctx context.Context) credentials.TransportCredentials {
	source := GetX509Source(ctx)
	tlsAuthorizer := tlsconfig.AuthorizeAny()
	return grpccredentials.MTLSClientCredentials(source, source, tlsAuthorizer)
}

func GetServer(ctx context.Context) credentials.TransportCredentials {
	source := GetX509Source(ctx)
	tlsAuthorizer := tlsconfig.AuthorizeAny()
	return grpccredentials.MTLSServerCredentials(source, source, tlsAuthorizer)
}

func GetServerWithSource(ctx context.Context, source *workloadapi.X509Source) credentials.TransportCredentials {
	tlsAuthorizer := tlsconfig.AuthorizeAny()
	return grpccredentials.MTLSServerCredentials(source, source, tlsAuthorizer)
}

func GetX509Source(ctx context.Context) *workloadapi.X509Source {
	// todo: retry if source in nil or empty
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		logrus.Errorf("error getting x509 source: %v", err)
	}
	var svid *x509svid.SVID
	svid, err = source.GetX509SVID()
	if err != nil {
		logrus.Errorf("error getting x509 svid: %v", err)
	}
	logrus.Infof("sVID: %q", svid.ID)
	return source
}
