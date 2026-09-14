package sks_cluster

import (
	"context"
	"encoding/base64"
	"errors"
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

// parseSKSNodepoolTaintV3 parses a CLI-formatted Kubernetes Node taint
// description formatted as VALUE:EFFECT, and returns discrete values
// for the value/effect as v3.SKSNodepoolTaint, or an error if
// the input value parsing failed.
func parseSKSNodepoolTaintV3(v string) (*v3.SKSNodepoolTaint, error) {
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("expected format VALUE:EFFECT")
	}
	taintValue, taintEffect := parts[0], parts[1]

	if taintValue == "" || taintEffect == "" {
		return nil, errors.New("expected format VALUE:EFFECT")
	}

	return &v3.SKSNodepoolTaint{
		Effect: v3.SKSNodepoolTaintEffect(taintEffect),
		Value:  taintValue,
	}, nil
}

// NVIDIA Multi-Instance GPU (MIG) profiles, keyed by the GPU instance type
// family the profile applies to. A nodepool has a single instance type (hence a
// single GPU family), so the family is inferred from `instance_type` rather than
// specified by the user. The accepted values mirror the Exoscale OpenAPI
// `nvidia-mig-profile-*` enums; update them if the spec gains new ones.
var sksNodepoolMIGProfileValues = map[v3.InstanceTypeFamily][]string{
	v3.InstanceTypeFamilyGpua30: {
		"2g.12gb", "1g.6gb+me", "1g.6gb", "2g.12gb+me", "4g.24gb",
	},
	v3.InstanceTypeFamilyGpurtx6000pro: {
		"1g.24gb-me", "1g.24gb", "2g.48gb-me", "2g.48gb", "4g.96gb+gfx",
		"1g.24gb+me", "2g.48gb+me.all", "1g.24gb+gfx", "1g.24gb+me.all",
		"4g.96gb", "2g.48gb+gfx",
	},
}

// sksNodepoolInstanceTypeFamily returns the (lowercased) family part of an
// `instance_type` value (`<family>.<size>`).
func sksNodepoolInstanceTypeFamily(instanceType string) string {
	parts := strings.SplitN(instanceType, ".", 2)
	return strings.ToLower(parts[0])
}

// sksNodepoolMIGProfiles builds the egoscale NvidiaMigProfiles for the given
// instance type family and MIG profile, inferring which GPU field to set from
// the family. It errors if the family is not MIG-capable or the profile is not
// valid for it.
func sksNodepoolMIGProfiles(family, profile string) (*v3.NvidiaMigProfiles, error) {
	values, ok := sksNodepoolMIGProfileValues[v3.InstanceTypeFamily(family)]
	if !ok {
		return nil, fmt.Errorf(
			"%s is only supported on NVIDIA MIG-capable instance types (families %q and %q), not %q",
			"nvidia_mig_profile",
			v3.InstanceTypeFamilyGpua30, v3.InstanceTypeFamilyGpurtx6000pro, family,
		)
	}

	if !in(values, profile) {
		return nil, fmt.Errorf(
			"unsupported MIG profile %q for instance type family %q; supported profiles are %s",
			profile, family, strings.Join(values, ", "),
		)
	}

	switch v3.InstanceTypeFamily(family) {
	case v3.InstanceTypeFamilyGpua30:
		return &v3.NvidiaMigProfiles{A3024gb: v3.NvidiaMigProfileA3024gb(profile)}, nil
	case v3.InstanceTypeFamilyGpurtx6000pro:
		return &v3.NvidiaMigProfiles{Rtxpro600096gb: v3.NvidiaMigProfileRtxpro600096gb(profile)}, nil
	default:
		// Unreachable: family presence was validated above.
		return nil, nil
	}
}

// sksNodepoolMIGProfile flattens the egoscale NvidiaMigProfiles into the single
// MIG profile value (whichever GPU family field is set), or "" when none is set.
func sksNodepoolMIGProfile(profiles *v3.NvidiaMigProfiles) string {
	if profiles == nil {
		return ""
	}
	if profiles.A3024gb != "" {
		return string(profiles.A3024gb)
	}
	if profiles.Rtxpro600096gb != "" {
		return string(profiles.Rtxpro600096gb)
	}

	return ""
}
