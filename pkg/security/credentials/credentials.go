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
	source := getX509Source()
	tlsAuthorizer := tlsconfig.AuthorizeAny()
	return grpccredentials.MTLSClientCredentials(source, source, tlsAuthorizer)
}

func GetServer(ctx context.Context) credentials.TransportCredentials {
	source := getX509Source()
	tlsAuthorizer := tlsconfig.AuthorizeAny()
	return grpccredentials.MTLSServerCredentials(source, source, tlsAuthorizer)
}

func getX509Source() *workloadapi.X509Source {
	source, err := workloadapi.NewX509Source(context.Background())
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
