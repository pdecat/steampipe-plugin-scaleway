package scaleway

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net/http"

	"github.com/scaleway/scaleway-sdk-go/api/k8s/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/turbot/steampipe-plugin-sdk/v6/plugin"
)

// getKubernetesClusterCertificateAuthority returns the certificate-authority-data of a cluster. The
// Kubernetes API exposes the cluster certificate authority nowhere but in the kubeconfig, so the
// kubeconfig is fetched with its legacy admin token redacted, and everything but the certificate
// authority is dropped.
func getKubernetesClusterCertificateAuthority(ctx context.Context, d *plugin.QueryData, h *plugin.HydrateData) (interface{}, error) {
	cluster := h.Item.(*k8s.Cluster)

	client, err := getSessionConfig(ctx, d)
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_kubernetes_cluster.getKubernetesClusterCertificateAuthority", "connection_error", err)
		return nil, err
	}

	kubeconfig, err := k8s.NewAPI(client).GetClusterKubeConfig(&k8s.GetClusterKubeConfigRequest{
		Region:    cluster.Region,
		ClusterID: cluster.ID,
		// Only the certificate authority is read: never fetch the legacy admin token.
		Redacted: scw.BoolPtr(true),
	}, scw.WithContext(ctx))
	if err != nil {
		plugin.Logger(ctx).Error("scaleway_kubernetes_cluster.getKubernetesClusterCertificateAuthority", "api_error", err, "id", cluster.ID)
		return nil, err
	}

	certificateAuthority, err := certificateAuthorityFromKubeconfig(kubeconfig)
	*kubeconfig = k8s.Kubeconfig{}
	if err != nil {
		plugin.Logger(ctx).Warn("scaleway_kubernetes_cluster.getKubernetesClusterCertificateAuthority", "error", err, "id", cluster.ID)
		return nil, nil
	}
	return certificateAuthority, nil
}

// certificateAuthorityFromKubeconfig returns the base64-encoded certificate-authority-data of the
// single cluster of a kubeconfig, after checking it decodes to a PEM certificate. Its errors never
// quote the kubeconfig.
func certificateAuthorityFromKubeconfig(kubeconfig *k8s.Kubeconfig) (string, error) {
	data, err := kubeconfig.GetCertificateAuthorityData()
	if err != nil {
		return "", errors.New("the kubeconfig does not hold exactly one cluster")
	}
	if data == "" {
		return "", errors.New("the kubeconfig has no certificate-authority-data")
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", errors.New("the certificate-authority-data is not valid base64")
	}
	if block, _ := pem.Decode(decoded); block == nil || block.Type != "CERTIFICATE" {
		return "", errors.New("the certificate-authority-data is not a PEM certificate")
	}
	return data, nil
}

// shouldIgnoreKubeconfigError reports whether reading the kubeconfig was refused (403) or found
// nothing (404): credentials limited to read-only permission sets may not read it, which must only
// leave the certificate_authority column null instead of failing the whole query.
func shouldIgnoreKubeconfigError(_ context.Context, _ *plugin.QueryData, _ *plugin.HydrateData, err error) bool {
	permissionsDeniedError := &scw.PermissionsDeniedError{}
	return errors.As(err, &permissionsDeniedError) || isHTTPCodeError(err, http.StatusForbidden) || is404Error(err)
}
