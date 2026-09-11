package sks_cluster

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// in returns true if v is found in list.
func in(list []string, v string) bool {
	for i := range list {
		if list[i] == v {
			return true
		}
	}

	return false
}

// defaultBool returns the value of the bool pointer v if not nil, otherwise the
// default value specified.
func defaultBool(v *bool, def bool) bool {
	if v != nil {
		return *v
	}

	return def
}

// optionalString returns a null types.String for an empty string, a value
// otherwise.
func optionalString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}

	return types.StringValue(s)
}

// appendAddon returns a new slice with addon appended.
func appendAddon(addons []string, addon string) []string {
	out := make([]string, 0, len(addons)+1)
	out = append(out, addons...)
	out = append(out, addon)

	return out
}

// removeAddon returns a new slice with addon removed.
func removeAddon(addons []string, addon string) []string {
	out := make([]string, 0, len(addons))
	for _, v := range addons {
		if v != addon {
			out = append(out, v)
		}
	}

	return out
}

// SKSClusterCertificates holds an SKS Cluster related CA certificates.
type SKSClusterCertificates struct {
	AggregationCA  string
	ControlPlaneCA string
	KubeletCA      string
}

// readClusterCertificates returns an SKS Cluster related CA certificates.
func readClusterCertificates(ctx context.Context, client *v3.Client, clusterID v3.UUID) (*SKSClusterCertificates, error) {
	encodedAggregationCertificate, err := client.GetSKSClusterAuthorityCert(ctx, clusterID, "aggregation")
	if err != nil {
		return nil, err
	}

	encodedControlPlaneCertificate, err := client.GetSKSClusterAuthorityCert(ctx, clusterID, "control-plane")
	if err != nil {
		return nil, err
	}

	encodedKubeletCertificate, err := client.GetSKSClusterAuthorityCert(ctx, clusterID, "kubelet")
	if err != nil {
		return nil, err
	}

	aggregationCertificate, err := base64.StdEncoding.DecodeString(encodedAggregationCertificate.Cacert)
	if err != nil {
		return nil, err
	}

	controlPlaneCertificate, err := base64.StdEncoding.DecodeString(encodedControlPlaneCertificate.Cacert)
	if err != nil {
		return nil, err
	}

	kubeletCertificate, err := base64.StdEncoding.DecodeString(encodedKubeletCertificate.Cacert)
	if err != nil {
		return nil, err
	}

	return &SKSClusterCertificates{
		AggregationCA:  string(aggregationCertificate),
		ControlPlaneCA: string(controlPlaneCertificate),
		KubeletCA:      string(kubeletCertificate),
	}, nil
}

// resolveSKSClusterVersion computes the major.minor.patch version of an SKS
// cluster from an inputVersion. Defaults to the latest version.
func resolveSKSClusterVersion(ctx context.Context, client *v3.Client, inputVersion string) (string, error) {
	inputVersionLength := len(strings.Split(inputVersion, "."))
	isMajorMinor := inputVersionLength == 2
	isMajorMinorPatch := inputVersionLength == 3

	if isMajorMinorPatch {
		return inputVersion, nil
	}

	availableVersions, err := client.ListSKSClusterVersions(ctx)
	if err != nil {
		return "", err
	}
	if len(availableVersions.SKSClusterVersions) == 0 {
		return "", fmt.Errorf("ListSKSClusterVersions: API returned empty list")
	}

	defaultVersion := availableVersions.SKSClusterVersions[0]

	if len(inputVersion) == 0 {
		return defaultVersion, nil
	}

	if isMajorMinor {
		for _, v := range availableVersions.SKSClusterVersions {
			if inputVersion == strings.Join(strings.Split(v, ".")[:2], ".") {
				return v, nil
			}
		}
		return "", fmt.Errorf("the SKS cluster version %s is not supported. Available versions: %s", inputVersion, strings.Join(availableVersions.SKSClusterVersions, ", "))
	}

	return "", fmt.Errorf("error resolving the provided SKS cluster version: %s. Available versions: %s", inputVersion, strings.Join(availableVersions.SKSClusterVersions, ", "))
}

// waitForClusterUpdateToSucceed works around a bug where the update operation
// can remain in pending state forever: it polls the cluster state instead.
func waitForClusterUpdateToSucceed(ctx context.Context, client *v3.Client, clusterID v3.UUID) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	hasStartedUpdate := false
	for {
		select {
		case <-ticker.C:
			cluster, err := client.GetSKSCluster(ctx, clusterID)
			if err != nil {
				return err
			}

			if hasStartedUpdate && cluster.State != "updating" {
				return nil
			} else if cluster.State == "updating" {
				hasStartedUpdate = true
			}
		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				return err
			}

			return nil
		}
	}
}

// await waits for the given operation to reach the success state.
func await(ctx context.Context, client *v3.Client) func(op *v3.Operation, err error) error {
	return func(op *v3.Operation, err error) error {
		if err != nil {
			return err
		}

		_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
		if err != nil {
			return err
		}

		return nil
	}
}

// sliceOrEmpty returns a non-nil slice so that types.SetValueFrom yields an
// empty set (rather than a null one) when the input is empty.
func sliceOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}

	return s
}
