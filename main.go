package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/indeedeng-alpha/harbor-container-webhook/internal/config"
	"github.com/indeedeng-alpha/harbor-container-webhook/internal/webhook"

	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
	_ = admissionv1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)

	var configPath string
	var kubeClientBurst int
	var kubeClientQPS float64
	var kubeClientlazyRemap bool
	flag.StringVar(&configPath, "config", "", "path to the config for the harbor-container-webhook")
	flag.IntVar(&kubeClientBurst, "kube-client-burst", rest.DefaultBurst, "Burst value for kubernetes client.")
	flag.Float64Var(&kubeClientQPS, "kube-client-qps", float64(rest.DefaultQPS), "QPS value for kubernetes client.")
	flag.BoolVar(&kubeClientlazyRemap, "kube-client-lazy-remap", false, "Deprecated. Has no effect.")
	flag.Parse()

	conf, err := config.LoadConfiguration(configPath)
	if err != nil {
		setupLog.Error(err, "unable to read config from "+configPath)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: conf.MetricsAddr,
		},
		WebhookServer: ctrlwebhook.NewServer(ctrlwebhook.Options{
			Port:    conf.Port,
			CertDir: conf.CertDir,
		}),
		HealthProbeBindAddress: conf.HealthAddr,
		LeaderElection:         false,
	})
	if err != nil {
		setupLog.Error(err, "unable to start harbor-container-webhook")
		os.Exit(1)
	}

	transformer, err := webhook.NewMultiTransformer(conf.Rules)
	if err != nil {
		setupLog.Error(err, "unable to start harbor-container-webhook")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("health-ping", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable add a liveness check to harbor-container-webhook")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("ready-ping", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable add a readiness check to harbor-container-webhook")
		os.Exit(1)
	}

	mutate := webhook.PodContainerProxier{
		Client:      mgr.GetClient(),
		Decoder:     admission.NewDecoder(scheme),
		Transformer: transformer,
		Verbose:     conf.Verbose,

		KubeClientQPS:   float32(kubeClientQPS),
		KubeClientBurst: kubeClientBurst,
	}
	setupLog.Info(fmt.Sprintf("kube client configured for %f.2 QPS, %d Burst", float32(kubeClientQPS), kubeClientBurst))

	mgr.GetWebhookServer().Register("/webhook-v1-pod", &ctrlwebhook.Admission{Handler: &mutate})

	setupLog.Info("starting harbor-container-webhook")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running harbor-container-webhook")
		os.Exit(1)
	}
}
