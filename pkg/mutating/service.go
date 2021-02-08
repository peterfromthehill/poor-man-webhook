package mutating

import (
	"encoding/json"
	"fmt"
	"log"
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

func (s ServiceMutate) createServicePort(config *config.Config) corev1.ServicePort {
	return corev1.ServicePort{
		Name:       config.ServicePort.Name,
		Port:       int32(config.ServicePort.Port),
		Protocol:   corev1.ProtocolTCP,
		TargetPort: utilv1.IntOrString{Type: utilv1.Int, IntVal: int32(config.ServicePort.Port)},
	}
}

func (s ServiceMutate) patch(service *corev1.Service, namespace string, config *config.Config) ([]byte, error) {
	//var ops []PatchOperation

	name := service.ObjectMeta.GetName()
	if name == "" {
		name = service.ObjectMeta.GetGenerateName()
	}

	servicePorts := service.Spec.Ports
	klog.Infof("ServiceSpec: %s", &service.Spec)
	if len(servicePorts) == 1 && servicePorts[0].Name == "" {
		servicePorts[0].Name = fmt.Sprintf("%s-%d", strings.ToLower(string(servicePorts[0].Protocol)), servicePorts[0].Port)
		log.Printf("set name from first servicePort: %s", servicePorts[0].Name)
	}

	isServicePortrdy := false
	for _, s := range servicePorts {
		if s.Port == int32(config.ServicePort.Port) {
			isServicePortrdy = true
		}
	}
	if !isServicePortrdy {
		servicePort := s.createServicePort(config)
		servicePorts = append(servicePorts, servicePort)
	}

	ops := []PatchOperation{
		PatchOperation{
			Op:    "replace",
			Path:  "/spec/ports",
			Value: servicePorts,
		},
	}
	ops = append(ops, addAnnotations(service.Annotations, map[string]string{AdmissionWebhookStatusKey: AdmissionWebhookStatusValue})...)

	return json.Marshal(ops)
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

	patchType := v1.PatchTypeJSONPatch
	klog.Info("Generated patch")
	return &v1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		UID:       request.UID,
		PatchType: &patchType,
	}
}
