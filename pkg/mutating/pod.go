package mutating

import (
	"encoding/json"
	"poor-man-webhook/pkg/config"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type PodMutate struct {
	MutateT
}

func PodMutating() *PodMutate {
	mutate := &PodMutate{
		MutateT{
			Path:   "/mutate/pod",
			Method: "POST",
		},
	}
	return mutate
}

func (p PodMutate) mkIptables(config *config.Config, podName, commonName, namespace string) (corev1.Container, error) {
	b := config.Iptables
	return b, nil
}

// mkProxy generates a new sidecar proxy based on the template provided in Config.
func (p PodMutate) mkProxy(config *config.Config, podName, commonName, namespace string) corev1.Container {
	r := config.Proxy
	r.ImagePullPolicy = corev1.PullAlways
	return r
}

func (p PodMutate) patch(pod *corev1.Pod, namespace string, config *config.Config) ([]byte, error) {
	var ops []PatchOperation

	name := pod.ObjectMeta.GetName()
	if name == "" {
		name = pod.ObjectMeta.GetGenerateName()
	}

	annotations := pod.ObjectMeta.GetAnnotations()
	commonName := annotations[AdmissionWebhookAnnotationKey]
	proxy := p.mkProxy(config, name, commonName, namespace)
	iptables, err := p.mkIptables(config, name, commonName, namespace)
	if err != nil {
		return nil, err
	}

	ops = append(ops, addContainers(pod.Spec.Containers, []corev1.Container{proxy}, "/spec/containers")...)
	ops = append(ops, addContainers(pod.Spec.InitContainers, []corev1.Container{iptables}, "/spec/initContainers")...)
	ops = append(ops, addAnnotations(pod.Annotations, map[string]string{AdmissionWebhookStatusKey: AdmissionWebhookStatusValue})...)
	return json.Marshal(ops)
}

// mutate takes an `AdmissionReview`, determines whether it is subject to mutation, and returns
// an appropriate `AdmissionResponse` including patches or any errors that occurred.
func (p PodMutate) Mutate(review *v1.AdmissionReview, config *config.Config) *v1.AdmissionResponse {
	request := review.Request
	var pod corev1.Pod
	if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
		klog.Error("Error unmarshaling pod")
		return &v1.AdmissionResponse{
			Allowed: false,
			UID:     request.UID,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	mutationAllowed, validationErr := p.ShouldMutate(&pod, config, request.Namespace, config.GetClusterDomain(), config.RestrictCertificatesToNamespace)

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

	patchBytes, err := p.patch(&pod, request.Namespace, config)
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
