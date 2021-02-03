package mutating

import (
	"poor-man-webhook/pkg/config"

	v1 "k8s.io/api/admission/v1"
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
)
