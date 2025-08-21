/*
Copyright 2021.

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

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"syscall"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/faisal-memon/sviddisk"
	meridiov1 "github.com/nordix/meridio/api/v1"
	meridiov1alpha1 "github.com/nordix/meridio/api/v1alpha1"
	attactorcontroller "github.com/nordix/meridio/pkg/controllers/attractor"
	"github.com/nordix/meridio/pkg/controllers/common"
	conduitcontroller "github.com/nordix/meridio/pkg/controllers/conduit"
	flowcontroller "github.com/nordix/meridio/pkg/controllers/flow"
	gatewaycontroller "github.com/nordix/meridio/pkg/controllers/gateway"
	streamcontroller "github.com/nordix/meridio/pkg/controllers/stream"
	trenchcontroller "github.com/nordix/meridio/pkg/controllers/trench"
	"github.com/nordix/meridio/pkg/debug"
	"github.com/nordix/meridio/pkg/log"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nordix/meridio/pkg/controllers/version"
	vipcontroller "github.com/nordix/meridio/pkg/controllers/vip"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func printHelp() {
	fmt.Println(`
nsp --
  The operator process in https://github.com/Nordix/Meridio
  This program shall be started in a Kubernetes container.`)
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(meridiov1alpha1.AddToScheme(scheme))
	utilruntime.Must(meridiov1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func setupTLSCert(socket string) error {
	ctx := context.Background()
	client, err := workloadapi.New(ctx, workloadapi.WithAddr(socket))
	if err != nil {
		return fmt.Errorf("unable to create workload API client: %w", err)
	}

	certDir := "/tmp/k8s-webhook-server/serving-certs"

	go func() {
		defer client.Close()
		err := client.WatchX509Context(ctx, &x509Watcher{CertDir: certDir})
		if err != nil && status.Code(err) != codes.Canceled {
			log.Fatal(setupLog, "error watching X.509 context", "error", err)
		}
	}()

	if err = sviddisk.WaitForCertificates(certDir); err != nil {
		return err
	}

	return nil
}

// x509Watcher is a sample implementation of the workloadapi.X509ContextWatcher interface
type x509Watcher struct {
	CertDir string
}

// UpdateX509SVIDs is run every time an SVID is updated
func (x *x509Watcher) OnX509ContextUpdate(c *workloadapi.X509Context) {
	err := sviddisk.WriteToDisk(c.DefaultSVID(), x.CertDir)
	if err != nil {
		setupLog.Error(err, "OnX509ContextUpdate")
	}
}

// OnX509ContextWatchError is run when the client runs into an error
func (x509Watcher) OnX509ContextWatchError(err error) {
	if status.Code(err) != codes.Canceled {
		setupLog.Error(err, "OnX509ContextWatchError")
	}
}

func main() {
	// Create a context that will be used to stop signal watching when the app exits
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var metricsAddr, probeAddr string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	if os.Getenv(common.LogLevelEnv) == "" { // trace as default value
		os.Setenv(common.LogLevelEnv, "TRACE")
	}

	ver := flag.Bool("version", false, "Print version and quit")
	debugCmd := flag.Bool("debug", false, "Print the debug information and quit")
	help := flag.Bool("help", false, "Print help and quit")

	flag.Parse()

	if *ver {
		fmt.Println(version.VersionInfo())
		os.Exit(0)
	}
	if *debugCmd {
		debug.MeridioVersion = version.VersionInfo()
		fmt.Println(debug.Collect().String())
		os.Exit(0)
	}
	if *help {
		printHelp()
		os.Exit(0)
	}

	logger := log.New("Operator", os.Getenv(common.LogLevelEnv))
	ctrl.SetLogger(logger)

	// Set up dynamic log level change via signals
	log.SetupLevelChangeOnSignal(ctx, map[os.Signal]string{
		syscall.SIGUSR1: common.LogLevelEnv,
		syscall.SIGUSR2: "TRACE",
	})

	// Set operator scope to the namespace where the operator pod exists
	// An empty value means the operator is running with cluster scope
	setupLog.Info(version.VersionInfo())
	namespace := os.Getenv("WATCH_NAMESPACE")
	if namespace == "" {
		setupLog.Info("operator is cluster-scoped")
	} else {
		setupLog.Info("operator is namespace-scoped", "namespace", namespace)
	}

	// Prepare tls cert when using spire
	var spiffeSocket string
	if spiffeSocket = os.Getenv("SPIFFE_ENDPOINT_SOCKET"); spiffeSocket != "" {
		setupLog.Info("using spire for webhook")
		if err := setupTLSCert(spiffeSocket); err != nil {
			setupLog.Error(err, "failed to setup the webhook")
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "132659e3.nordix.org",
		WebhookServer: &webhook.DefaultServer{
			Options: webhook.Options{
				Port: 9443,
				TLSOpts: []func(*tls.Config){
					func(c *tls.Config) {
						c.MinVersion = tls.VersionTLS12
					},
				},
			},
		},
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				namespace: {},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&trenchcontroller.TrenchReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Trench"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Trench")
		os.Exit(1)
	}
	if err = (&meridiov1.Trench{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Trench")
		os.Exit(1)
	}

	if err = (&vipcontroller.VipReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Vip"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Vip")
		os.Exit(1)
	}
	if err = (&meridiov1.Vip{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Vip")
		os.Exit(1)
	}

	if err = (&attactorcontroller.AttractorReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Attractor"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Attractor")
		os.Exit(1)
	}
	if err = (&meridiov1.Attractor{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Attractor")
		os.Exit(1)
	}

	if err = (&gatewaycontroller.GatewayReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Gateway"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Gateway")
		os.Exit(1)
	}
	if err = (&meridiov1.Gateway{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Gateway")
		os.Exit(1)
	}

	if err = (&conduitcontroller.ConduitReconciler{
		Client:    mgr.GetClient(),
		APIReader: mgr.GetAPIReader(),
		Log:       ctrl.Log.WithName("controllers").WithName("Conduit"),
		Scheme:    mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Conduit")
		os.Exit(1)
	}
	if err = (&meridiov1.Conduit{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Conduit")
		os.Exit(1)
	}

	if err = (&streamcontroller.StreamReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Stream"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Stream")
		os.Exit(1)
	}
	if err = (&meridiov1.Stream{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Stream")
		os.Exit(1)
	}

	if err = (&flowcontroller.FlowReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Flow"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Flow")
		os.Exit(1)
	}
	if err = (&meridiov1.Flow{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Flow")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
