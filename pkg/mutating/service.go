package mutating

import (
	"encoding/json"
	"fmt"
	"poor-man-webhook/pkg/config"
	"strings"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilv1 "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
)

type ServiceMutate struct {
	MutateT
}

func ServiceMutating() *ServiceMutate {
	mutate := &ServiceMutate{
		MutateT{
			Path:   "/mutate/service",
			Method: "POST",
		},
	}
	return mutate
}

func (s ServiceMutate) mkIptables(config *config.Config, podName, commonName, namespace string) (corev1.Container, error) {
	b := config.Iptables
	return b, nil
}

// mkProxy generates a new sidecar proxy based on the template provided in Config.
func (s ServiceMutate) mkProxy(config *config.Config, podName, commonName, namespace string) corev1.Container {
	r := config.Proxy
	r.ImagePullPolicy = corev1.PullAlways
	return r
}

func (s ServiceMutate) createServicePort(config *config.Config) corev1.ServicePort {
	return corev1.ServicePort{
		Name:       config.ServicePort.Name,
		Port:       int32(config.ServicePort.Port),
		Protocol:   corev1.ProtocolTCP,
		TargetPort: utilv1.IntOrString{Type: utilv1.Int, IntVal: int32(config.ServicePort.Port)},
	}
}

func (s ServiceMutate) fixOnePortServices(servicePort corev1.ServicePort) corev1.ServicePort {
	sdc := servicePort.DeepCopy()
	sdc.Name = fmt.Sprintf("%s-%d", strings.ToLower(string(sdc.Protocol)), sdc.Port)
	return *sdc
}

func (s ServiceMutate) addServicePort(existing, new []corev1.ServicePort, path string) (ops []PatchOperation) {
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

func (s ServiceMutate) replaceServicePorts(existing, new []corev1.ServicePort, path string) (ops []PatchOperation) {
	return []PatchOperation{
		{
			Op:    "replace",
			Path:  path,
			Value: new,
		},
	}
}

func (s ServiceMutate) patch(service *corev1.Service, namespace string, config *config.Config) ([]byte, error) {
	var ops []PatchOperation

	name := service.ObjectMeta.GetName()
	if name == "" {
		name = service.ObjectMeta.GetGenerateName()
	}

	//services := []corev1.ServicePort{}
	klog.Infof("ServiceSpec: %s", &service.Spec)
	if len(service.Spec.Ports) == 1 && service.Spec.Ports[0].Name == "" {
		klog.Infof("just one service and without a name!")
		servicePort := service.Spec.Ports[0]
		servicePortName := fmt.Sprintf("%s-%d", strings.ToLower(string(servicePort.Protocol)), servicePort.Port)
		op := PatchOperation{
			Op:    "replace",
			Path:  "/spec/ports/0/name",
			Value: servicePortName,
		}
		klog.Infof("OP: %q", op)
		ops = append(ops, op)
	}
	servicePort := s.createServicePort(config)

	op := PatchOperation{
		Op:    "add",
		Path:  "/spec/ports/-",
		Value: servicePort,
	}
	ops = append(ops, op)

	return json.Marshal(ops)
}

func (s ServiceMutate) ShouldMutate(service *corev1.Service, config *config.Config, namespace string, clusterDomain string, restrictToNamespace bool) (bool, error) {
	annotations := service.ObjectMeta.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	// Only mutate if the object is annotated appropriately (annotation key set) and we haven't
	// mutated already (status key isn't set).
	// if annotations[admissionWebhookAnnotationKey] == "" || annotations[admissionWebhookStatusKey] == "injected" {
	// 	return false, nil
	// }

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

// mutate takes an `AdmissionReview`, determines whether it is subject to mutation, and returns
// an appropriate `AdmissionResponse` including patches or any errors that occurred.
func (s ServiceMutate) Mutate(review *v1.AdmissionReview, config *config.Config) *v1.AdmissionResponse {
	request := review.Request
	var service corev1.Service
	if err := json.Unmarshal(request.Object.Raw, &service); err != nil {
		klog.Error("Error unmarshaling pod")
		return &v1.AdmissionResponse{
			Allowed: false,
			UID:     request.UID,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	mutationAllowed, validationErr := s.ShouldMutate(&service, config, request.Namespace, config.GetClusterDomain(), config.RestrictCertificatesToNamespace)
	klog.Infof("should we mutate? %s / ValiErr: %v", mutationAllowed, validationErr)
	if validationErr != nil {
		klog.Info("Validation error")
		return &v1.AdmissionResponse{
			Allowed: false,
			UID:     request.UID,
			Result: &metav1.Status{
				Message: validationErr.Error(),
			},
		}
	}

	if !mutationAllowed {
		klog.Info("Skipping mutation")
		return &v1.AdmissionResponse{
			Allowed: true,
			UID:     request.UID,
		}
	}

	patchBytes, err := s.patch(&service, request.Namespace, config)
	if err != nil {
		klog.Error("Error generating patch")
		return &v1.AdmissionResponse{
			Allowed: false,
			UID:     request.UID,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	klog.Info("Generated patch")
	return &v1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		UID:     request.UID,
		PatchType: func() *v1.PatchType {
			pt := v1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}
