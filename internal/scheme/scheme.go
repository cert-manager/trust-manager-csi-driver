package scheme

import (
	"github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata"
	metadatav1alpha1 "github.com/cert-manager/trust-manager-csi-driver/internal/api/metadata/v1alpha1"
	trustv1alpha1 "github.com/cert-manager/trust-manager/pkg/apis/trust/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	kubernetes "k8s.io/client-go/kubernetes/scheme"
)

func New() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = metadata.AddToScheme(scheme)
	_ = metadatav1alpha1.AddToScheme(scheme)
	_ = trustv1alpha1.AddToScheme(scheme)
	_ = kubernetes.AddToScheme(scheme)
	return scheme
}
