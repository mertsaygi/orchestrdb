package main

import (
	"flag"
	"os"

	v1alpha1 "github.com/mertsaygi/orchestrdb/src/api/v1alpha1"
	"github.com/mertsaygi/orchestrdb/src/controllers"
	"github.com/mertsaygi/orchestrdb/src/db"
	"github.com/mertsaygi/orchestrdb/src/services"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	var enableLeaderElection bool

	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.Parse()

	// Configure logger
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// NOTE: In controller-runtime v0.18.x, Options does NOT have MetricsBindAddress or Port fields.
	// We keep it minimal here.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:         scheme,
		LeaderElection: enableLeaderElection,
		// This ID should be unique within the cluster.
		LeaderElectionID: "orchestrdb-operator.mertsaygi.net",
	})
	if err != nil {
		ctrl.Log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Wire DB adapter + service
	postgresAdapter := db.NewPostgresAdapter()
	dbService := services.NewDatabaseService(postgresAdapter)

	// Register controller
	if err = (&controllers.DatabaseReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		Log:             ctrl.Log.WithName("controllers").WithName("Database"),
		DatabaseService: dbService,
	}).SetupWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "unable to create controller", "controller", "Database")
		os.Exit(1)
	}

	ctrl.Log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		ctrl.Log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
