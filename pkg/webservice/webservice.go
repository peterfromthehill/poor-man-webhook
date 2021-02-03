package webservice

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"poor-man-webhook/pkg/config"
	"poor-man-webhook/pkg/mutating"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/peterfromthehill/autocertLego"
	"golang.org/x/crypto/acme/autocert"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
)

func newRouter(config *config.Config) http.Handler {
	klog.Info("Creating Router")
	r := mux.NewRouter()
	podMutater := mutating.PodMutating()
	r.HandleFunc(podMutater.Path, mutate(config, podMutater.Mutate)).Methods(podMutater.Method)
	serviceMutator := mutating.ServiceMutating()
	r.HandleFunc(serviceMutator.Path, mutate(config, serviceMutator.Mutate)).Methods(serviceMutator.Method)
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	return loggedRouter
	//return r
}

func CreateHttpServer(customCAPath string, config *config.Config) *http.Server {
	whitelistedDomains := []string{
		fmt.Sprintf("%s.%s.svc.%s", config.Service, config.Namespace, config.ClusterDomain),
		fmt.Sprintf("%s.%s.svc", config.Service, config.Namespace),
		fmt.Sprintf("%s.%s", config.Service, config.Namespace),
	}

	klog.Infof("Whitelisted Domains: %q", whitelistedDomains)

	manager := &autocertLego.Manager{
		EMail:      config.EMail,
		Directory:  config.CAUrl,
		HostPolicy: autocertLego.HostWhitelist(whitelistedDomains...),
		DirCache:   autocert.DirCache("./secret-dir/"),
	}

	srv := &http.Server{
		Addr:      config.GetAddress(),
		TLSConfig: manager.TLSConfig(),
		Handler:   newRouter(config),
	}

	return srv
}

func mutate(config *config.Config, fmutate func(review *v1.AdmissionReview, config *config.Config) *v1.AdmissionResponse) func(w http.ResponseWriter, r *http.Request) {
	klog.Info("Init handler")
	return func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil {
			if data, err := ioutil.ReadAll(r.Body); err == nil {
				body = data
			}
		}
		if len(body) == 0 {
			klog.Error("Bad Request: 400 (Empty Body)")
			http.Error(w, "Bad Request (Empty Body)", http.StatusBadRequest)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			klog.Error("Bad Request: 415 (Unsupported Media Type)")
			http.Error(w, fmt.Sprintf("Bad Request: 415 Unsupported Media Type (Expected Content-Type 'application/json' but got '%s')", contentType), http.StatusUnsupportedMediaType)
			return
		}
		var response *v1.AdmissionResponse
		review := v1.AdmissionReview{}
		runtimeScheme := runtime.NewScheme()
		codecs := serializer.NewCodecFactory(runtimeScheme)
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(body, nil, &review); err != nil {
			klog.Error("Can't decode body")
			response = &v1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		} else {
			response = fmutate(&review, config)
		}
		resp, err := json.Marshal(v1.AdmissionReview{
			Response: response,
		})
		if err != nil {
			klog.Info("Marshal error")
			http.Error(w, fmt.Sprintf("Marshal Error: %v", err), http.StatusInternalServerError)
		} else {
			klog.Info("Returning review")
			if _, err := w.Write(resp); err != nil {
				klog.Info("Write error")
			}
		}
	}
}
