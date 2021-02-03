package main

import (
	"flag"
	"poor-man-webhook/pkg/certpool"
	"poor-man-webhook/pkg/config"
	"poor-man-webhook/pkg/webservice"

	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

const (
	admissionWebhookAnnotationKey = "pmsm/name"
	admissionWebhookStatusKey     = "pmsm/status"
)

type ArgsConfig struct {
	configfile string
	cacert     string
	directory  string
}

var argsConfig *ArgsConfig

func init() {
	argsConfig = &ArgsConfig{}
	flag.StringVar(&argsConfig.configfile, "configfile", "config.json", "config")
	flag.StringVar(&argsConfig.cacert, "cacert", "/ca/tls.crt", "cacert")
}

func main() {
	flag.Parse()
	config, err := config.LoadConfig(argsConfig.configfile)
	if err != nil {
		panic(err)
	}
	certpool.GetCache().AddCerts(argsConfig.cacert)
	srv := webservice.CreateHttpServer(argsConfig.cacert, config)
	klog.Info("Listening on ", config.GetAddress(), "...")
	panic(srv.ListenAndServeTLS("", ""))
}

// "https://acme.proxy.svc.cluster.local/acme/development/directory"
// "poor-man-hook.proxy", "poor-man-hook", "poor-man-hook.proxy.svc", "poor-man-hook.proxy.svc.cluster.local"
// package main

// import (
// 	"log"
// 	"net/http"
// 	"poor-man-webhook/pkg/certpool"

// 	"github.com/foomo/simplecert"
// 	"github.com/foomo/tlsconfig"
// )

// func main() {
// 	certpool.GetCache().AddCerts("ca.crt")
// 	// tlsClientConfig := &tls.Config{
// 	// 	RootCAs:            certpool.GetCache().RootCAs,
// 	// 	InsecureSkipVerify: true,
// 	// }
// 	// tr := &http.Transport{TLSClientConfig: tlsClientConfig}
// 	// client := &http.Client{Transport: tr}

// 	cfg := simplecert.Default
// 	cfg.Domains = []string{"poor-man-hook.proxy", "poor-man-hook.proxy.svc", "poor-man-hook.proxy.svc.cluster.local"}
// 	cfg.CacheDir = "."
// 	cfg.DirectoryURL = "https://acme.unusual.one/acme/development/directory"
// 	cfg.SSLEmail = "you@emailprovider.com"
// 	certReloader, err := simplecert.Init(cfg, nil)
// 	if err != nil {
// 		log.Fatal("simplecert init failed: ", err)
// 	}

// 	// redirect HTTP to HTTPS
// 	// CAUTION: This has to be done AFTER simplecert setup
// 	// Otherwise Port 80 will be blocked and cert registration fails!
// 	log.Println("starting HTTP Listener on Port 80")
// 	go http.ListenAndServe(":80", http.HandlerFunc(simplecert.Redirect))
// 	// init strict tlsConfig with certReloader
// 	// you could also use a default &tls.Config{}, but be warned this is highly insecure
// 	tlsconf := tlsconfig.NewServerTLSConfig(tlsconfig.TLSModeServerStrict)

// 	// now set GetCertificate to the reloaders GetCertificateFunc to enable hot reload
// 	tlsconf.GetCertificate = certReloader.GetCertificateFunc()

// 	// init server
// 	s := &http.Server{
// 		Addr:      ":443",
// 		TLSConfig: tlsconf,
// 	}

// 	// lets go
// 	log.Fatal(s.ListenAndServeTLS("", ""))
// }
