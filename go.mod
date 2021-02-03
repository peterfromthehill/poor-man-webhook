module poor-man-webhook

go 1.15

require (
	github.com/foomo/tlsconfig v0.0.0-20180418120404-b67861b076c9 // indirect
	github.com/go-acme/lego/v4 v4.2.0 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/peterfromthehill/autocertLego v0.0.0-20210207150939-de747caffd7a
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777 // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/klog/v2 v2.5.0
)

// Pin k8s deps to 1.20.2
replace (
	k8s.io/api => k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.2
	k8s.io/apiserver => k8s.io/apiserver v0.20.2
	k8s.io/client-go => k8s.io/client-go v0.20.2
	k8s.io/code-generator => k8s.io/code-generator v0.20.2
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200410145947-bcb3869e6f29
)
