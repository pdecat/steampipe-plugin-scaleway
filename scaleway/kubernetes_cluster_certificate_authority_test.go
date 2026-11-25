package scaleway

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/scaleway/scaleway-sdk-go/api/k8s/v1"
	sdkerrors "github.com/scaleway/scaleway-sdk-go/errors"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// testCertificateAuthority is a throwaway self-signed certificate (CN=test-ca) whose key was discarded.
const testCertificateAuthority = "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJlVENDQVIrZ0F3SUJBZ0lVZC9VbUhNVmRlWUhTd0MxRUwwVnhoNGZaTlZ3d0NnWUlLb1pJemowRUF3SXcKRWpFUU1BNEdBMVVFQXd3SGRHVnpkQzFqWVRBZUZ3MHlOakV4TVRZeE9EVTFORGxhRncwek5qRXhNVE14T0RVMQpORGxhTUJJeEVEQU9CZ05WQkFNTUIzUmxjM1F0WTJFd1dUQVRCZ2NxaGtqT1BRSUJCZ2dxaGtqT1BRTUJCd05DCkFBVDE1bzRwSnZENW8vZ1lPNFVwOFp0eU5KUWxPL2NSK3Z2aVFpV3ZIR29rRzBybDUwUG9yZkFwZXRVM0hWUVQKT05DOHh0c0E5dkJ4ODFraVdKRkJ0d3N2bzFNd1VUQWRCZ05WSFE0RUZnUVVHbUp1bGJJbHA4cXNpVyt2UWczQQo0K0Fob2F3d0h3WURWUjBqQkJnd0ZvQVVHbUp1bGJJbHA4cXNpVyt2UWczQTQrQWhvYXd3RHdZRFZSMFRBUUgvCkJBVXdBd0VCL3pBS0JnZ3Foa2pPUFFRREFnTklBREJGQWlBS1lLb3pnQ1FsSFYwdExSUGdVVGJCeHJ5WElES0wKNU1WRHJrWlllejNZTGdJaEFQVm8vUC9TbDlId3krUjhYWjgxV1ZldFdRUEo4RlFlUkdkVUpJV1QvSFdXCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"

func testKubeconfig(certificateAuthorityData ...string) *k8s.Kubeconfig {
	kubeconfig := &k8s.Kubeconfig{
		APIVersion: "v1",
		Kind:       "Config",
		Users: []*k8s.KubeconfigUserWithName{
			{Name: "test-admin", User: k8s.KubeconfigUser{Token: "token-placeholder"}},
		},
	}
	for _, data := range certificateAuthorityData {
		kubeconfig.Clusters = append(kubeconfig.Clusters, &k8s.KubeconfigClusterWithName{
			Name: "test",
			Cluster: k8s.KubeconfigCluster{
				Server:                   "https://00000000-0000-0000-0000-000000000000.api.k8s.fr-par.scw.cloud:6443",
				CertificateAuthorityData: data,
			},
		})
	}
	return kubeconfig
}

func TestCertificateAuthorityFromKubeconfig(t *testing.T) {
	got, err := certificateAuthorityFromKubeconfig(testKubeconfig(testCertificateAuthority))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != testCertificateAuthority {
		t.Fatalf("got %q, want the certificate-authority-data", got)
	}
}

func TestCertificateAuthorityFromKubeconfigRejects(t *testing.T) {
	notACertificate := base64.StdEncoding.EncodeToString([]byte("-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"))
	for name, kubeconfig := range map[string]*k8s.Kubeconfig{
		"no clusters":       testKubeconfig(),
		"two clusters":      testKubeconfig(testCertificateAuthority, testCertificateAuthority),
		"no ca data":        testKubeconfig(""),
		"not base64":        testKubeconfig("not-base64!"),
		"not a certificate": testKubeconfig(notACertificate),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := certificateAuthorityFromKubeconfig(kubeconfig)
			if err == nil {
				t.Fatalf("expected an error, got %q", got)
			}
			// an error must never quote the kubeconfig
			for _, content := range []string{"placeholder", "api.k8s", "LS0tLS1", "AAAA"} {
				if strings.Contains(err.Error(), content) {
					t.Fatalf("error quotes the kubeconfig: %v", err)
				}
			}
		})
	}
}

func TestShouldIgnoreKubeconfigError(t *testing.T) {
	// GetClusterKubeConfig wraps every API error, so the check must see through the wrapping
	wrap := func(err error) error { return sdkerrors.Wrap(err, "error getting cluster kubeconfig") }
	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"permissions denied": {wrap(&scw.PermissionsDeniedError{}), true},
		"forbidden":          {wrap(&scw.ResponseError{StatusCode: http.StatusForbidden}), true},
		"not found":          {wrap(&scw.ResourceNotFoundError{}), true},
		"http not found":     {wrap(&scw.ResponseError{StatusCode: http.StatusNotFound}), true},
		"unauthenticated":    {wrap(&scw.ResponseError{StatusCode: http.StatusUnauthorized}), false},
		"server error":       {wrap(&scw.ResponseError{StatusCode: http.StatusInternalServerError}), false},
		"other":              {wrap(fmt.Errorf("boom")), false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := shouldIgnoreKubeconfigError(context.Background(), nil, nil, tc.err); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
