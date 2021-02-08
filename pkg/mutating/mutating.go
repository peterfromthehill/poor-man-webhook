package mutating

import (
	"fmt"
	"poor-man-webhook/pkg/config"
	"strconv"
	"strings"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MutateT struct {
	Path   string
	Method string
	Mutate func(review *v1.AdmissionReview, config *config.Config) *v1.AdmissionResponse
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

const (
	AdmissionWebhookAnnotationKey = "pmsm/name"
	AdmissionWebhookStatusKey     = "pmsm/status"
	AdmissionWebhookStatusValue   = "injected"
	AdmissionWebhookIgnoreKey     = "pmsm/ignore"
)

func (s MutateT) ShouldMutate(object interface{}, config *config.Config, namespace string, clusterDomain string, restrictToNamespace bool) (bool, error) {
	var pod corev1.Pod
	var service corev1.Service
	var objectMeta metav1.ObjectMeta
	switch o := object.(type) {
	case corev1.Service:
		service = o
		objectMeta = service.ObjectMeta
	case corev1.Pod:
		pod = o
		objectMeta = pod.ObjectMeta
	}

	annotations := objectMeta.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	if annotations[AdmissionWebhookIgnoreKey] != "" {
		b, err := strconv.ParseBool(annotations[AdmissionWebhookIgnoreKey])
		return !b, err
	}

	if annotations[AdmissionWebhookStatusKey] == "injected" {
		return false, nil
	}

	for _, p := range service.Spec.Ports {
		if p.Port == int32(config.ServicePort.Port) {
			// Port bereits vorhanden
			return false, nil
		}
	}

	if !restrictToNamespace {
		return true, nil
	}

	subject := strings.Trim(annotations[AdmissionWebhookAnnotationKey], ".")

	err := fmt.Errorf("subject \"%s\" matches a namespace other than \"%s\" and is not permitted. This check can be disabled by setting restrictCertificatesToNamespace to false in the autocert-config ConfigMap", subject, namespace)

	if strings.HasSuffix(subject, ".svc") && !strings.HasSuffix(subject, fmt.Sprintf(".%s.svc", namespace)) {
		return false, err
	}

	if strings.HasSuffix(subject, fmt.Sprintf(".svc.%s", clusterDomain)) && !strings.HasSuffix(subject, fmt.Sprintf(".%s.svc.%s", namespace, clusterDomain)) {
		return false, err
	}

	return true, nil
}

func addAnnotations(existing, new map[string]string) (ops []PatchOperation) {
	if len(existing) == 0 {
		return []PatchOperation{
			{
				Op:    "add",
				Path:  "/metadata/annotations",
				Value: new,
			},
		}
	}
	for k, v := range new {
		if existing[k] == "" {
			ops = append(ops, PatchOperation{
				Op:    "add",
				Path:  "/metadata/annotations/" + escapeJSONPath(k),
				Value: v,
			})
		} else {
			ops = append(ops, PatchOperation{
				Op:    "replace",
				Path:  "/metadata/annotations/" + escapeJSONPath(k),
				Value: v,
			})
		}
	}
	return ops
}

func addContainers(existing, new []corev1.Container, path string) (ops []PatchOperation) {
	if len(existing) == 0 {
		return []PatchOperation{
			{
				Op:    "add",
				Path:  path,
				Value: new,
			},
		}
	}

	for _, add := range new {
		ops = append(ops, PatchOperation{
			Op:    "add",
			Path:  path + "/-",
			Value: add,
		})
	}
	return ops
}

// RFC6901 JSONPath Escaping -- https://tools.ietf.org/html/rfc6901
func escapeJSONPath(path string) string {
	// Replace`~` with `~0` then `/` with `~1`. Note that the order
	// matters otherwise we'll turn a `/` into a `~/`.
	path = strings.Replace(path, "~", "~0", -1)
	path = strings.Replace(path, "/", "~1", -1)
	return path
}
