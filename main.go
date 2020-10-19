/*
Copyright 2020 Indeed.
*/

package main

import (
	"flag"
	"os"

	"indeed.com/devops-incubation/harbor-container-webhook/internal/config"
	"indeed.com/devops-incubation/harbor-container-webhook/internal/dynamic"
	"indeed.com/devops-incubation/harbor-container-webhook/internal/mutate"
	"indeed.com/devops-incubation/harbor-container-webhook/internal/static"

	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
	"sigs.k8s.io/controller-runtime/pkg/webhook"
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
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to the config for the harbor-container-webhook")
	flag.Parse()

	conf, err := config.LoadConfiguration(configPath)
	if err != nil {
		setupLog.Error(err, "unable to read config from "+configPath)
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: conf.MetricsAddr,
		Port:               conf.Port,
		LeaderElection:     conf.EnableLeaderElection,
		LeaderElectionID:   "harbor-container-webhook",
		CertDir:            conf.CertDir,
	})
	if err != nil {
		setupLog.Error(err, "unable to start harbor-container-webhook")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	var transformer mutate.ContainerTransformer
	if conf.Dynamic.Enabled {
		transformer = dynamic.NewTransformer(conf.Dynamic)
	} else {
		transformer = static.NewTransformer(conf.Static)
	}

	decoder, _ := admission.NewDecoder(scheme)
	mutate := mutate.PodContainerProxier{
		Client:      mgr.GetClient(),
		Decoder:     decoder,
		Transformer: transformer,
	}

	mgr.GetWebhookServer().Register("/mutate-v1-pod", &webhook.Admission{Handler: &mutate})

	setupLog.Info("starting harbor-container-webhook")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running harbor-container-webhook")
		os.Exit(1)
	}
}
