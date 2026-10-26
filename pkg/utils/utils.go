package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
)

type IDStringer interface {
	Id() string
}

func IDString(d IDStringer, name string) string {
	id := d.Id()
	if id == "" {
		id = "<new resource>"
	}

	return fmt.Sprintf("%s (ID = %s)", name, id)
}

// In returns true if v is found in list.
func In(list []string, v string) bool {
	for i := range list {
		if list[i] == v {
			return true
		}
	}

	return false
}

// DefaultString returns the value of the string pointer v if not nil, otherwise the default value specified.
func DefaultString(v *string, def string) string {
	if v != nil {
		return *v
	}

	return def
}

// DefaultInt64 returns the value of the int64 pointer v if not nil, otherwise the default value specified.
func DefaultInt64(v *int64, def int64) int64 {
	if v != nil {
		return *v
	}

	return def
}

// DefaultBool returns the value of the bool pointer v if not nil, otherwise the default value specified.
func DefaultBool(v *bool, def bool) bool {
	if v != nil {
		return *v
	}

	return def
}

// AddressToStringPtr returns a string representation of addr if not nil, otherwise nil.
func AddressToStringPtr(addr *net.IP) *string {
	if addr != nil {
		addrStr := addr.String()
		return &addrStr
	}

	return nil
}

// NonEmptyStringPtr returns a non-nil pointer to s if the string is not empty, otherwise nil.
func NonEmptyStringPtr(s string) *string {
	if s != "" {
		return &s
	}

	return nil
}

func Unique(s []string) []string {
	inResult := map[string]struct{}{}
	var result []string
	for _, str := range s {
		if _, ok := inResult[str]; !ok {
			inResult[str] = struct{}{}
			result = append(result, str)
		}
	}
	return result
}

// EncodeUserData does compression and base64 encoding, used in resource_exoscale_compute[_instance[_pool]]
// returns (user_data, user_data_already_base64, error)
func EncodeUserData(userData string) (string, bool, error) {
	// template_cloudinit_config alows to gzip but not base64, prevent such case
	if len(userData) > 2 && userData[0] == '\x1f' && userData[1] == '\x8b' {
		return "", false, errors.New("user_data appears to be gzipped: it should be left raw, or also be base64 encoded")
	}

	// If user supplied data is already base64 encoded, do nothing.
	_, err := base64.StdEncoding.DecodeString(userData)
	if err == nil {
		return userData, true, nil
	}

	userDataBase64 := base64.StdEncoding.EncodeToString([]byte(userData))

	if len(userDataBase64) >= config.ComputeMaxUserDataLength {
		return "", false, fmt.Errorf("user-data maximum allowed length is %d bytes", config.ComputeMaxUserDataLength)
	}

	return userDataBase64, false, nil
}

// DecodeUserData does base64 decoding & decompression, used in resource_exoscale_compute[_instance[_pool]]
func DecodeUserData(data string) (string, error) {
	b64Decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}

	gz, err := gzip.NewReader(bytes.NewReader(b64Decoded))
	if err != nil {
		if errors.Is(err, gzip.ErrHeader) {
			// User data are not compressed, returning as-is.
			return string(b64Decoded), nil
		}

		return "", err
	}
	defer gz.Close()

	userData, err := io.ReadAll(gz)
	if err != nil {
		return "", err
	}

	return string(userData), nil
}

// ValidateLowercaseString validates that the given fields contains only lowercase characters
func ValidateLowercaseString(val any, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.ContainsAny(v, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		errs = append(errs, fmt.Errorf("%q must be lowercase, got: %q", key, v))
	}
	return
}

// SwitchClientZone clones the existing exoscale Client in the new zone.
func SwitchClientZone(ctx context.Context, client *exoscale.Client, zone exoscale.ZoneName) (*exoscale.Client, error) {
	if zone == "" {
		return client, nil
	}
	endpoint, err := client.GetZoneAPIEndpoint(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("switch client zone v3: %w", err)
	}

	return client.WithEndpoint(endpoint), nil
}

// FindInstanceTypeByName copies the behaviour of egoscale v2's client.FindInstanceType
// but using the v3 api:
// FindInstanceType attempts to find an Instance type by family+size or ID.
// To search by family+size, the expected format for v is "[FAMILY.]SIZE" (e.g. "large", "gpu.medium"),
// with family defaulting to "standard" if not specified.
func FindInstanceTypeByNameV3(ctx context.Context, client *exoscale.Client, id string) (*exoscale.InstanceType, error) {

	var typeFamily, typeSize string

	parts := strings.SplitN(id, ".", 2)
	if l := len(parts); l > 0 {
		if l == 1 {
			typeFamily, typeSize = "standard", strings.ToLower(parts[0])
		} else {
			typeFamily, typeSize = strings.ToLower(parts[0]), strings.ToLower(parts[1])
		}
	}

	res, err := client.ListInstanceTypes(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range res.InstanceTypes {
		if string(r.Family) == typeFamily && string(r.Size) == typeSize {
			return client.GetInstanceType(ctx, r.ID)
		}
	}

	return nil, exoscale.ErrNotFound
}

// In the context of converting list of IDs to resources for egoscale V3 update and
// creation requests, we're using a  lot of inline funcitons, increasing reading complexity
// when one is called more than once, we should aim to regroup them here.
func AntiAffiniGroupsToAntiAffinityGroupIDs(aags []exoscale.AntiAffinityGroup) (ls []string) {
	ls = make([]string, len(aags))
	for i, aag := range aags {
		ls[i] = aag.ID.String()
	}
	return
}

func AntiAffinityGroupIDsToAntiAffinityGroups(ids []any) (ls []exoscale.AntiAffinityGroup) {
	ls = make([]exoscale.AntiAffinityGroup, len(ids))
	for i, id := range ids {
		ls[i] = exoscale.AntiAffinityGroup{
			ID: exoscale.UUID(id.(string)),
		}
	}
	return
}

func PrivateNetworksToPrivateNetworkIDs(privnets []exoscale.PrivateNetwork) (ls []string) {
	ls = make([]string, len(privnets))
	for i, aag := range privnets {
		ls[i] = aag.ID.String()
	}
	return
}

func PrivateNetworkIDsToPrivateNetworks(ids []any) (ls []exoscale.PrivateNetwork) {
	ls = make([]exoscale.PrivateNetwork, len(ids))
	for i, id := range ids {
		ls[i] = exoscale.PrivateNetwork{
			ID: exoscale.UUID(id.(string)),
		}
	}
	return
}

func SecurityGroupsToSecurityGroupIDs(sgs []exoscale.SecurityGroup) (ls []string) {
	ls = make([]string, len(sgs))
	for i, aag := range sgs {
		ls[i] = aag.ID.String()
	}
	return
}

func SecurityGroupIDsToSecurityGroups(ids []any) (ls []exoscale.SecurityGroup) {
	ls = make([]exoscale.SecurityGroup, len(ids))
	for i, id := range ids {
		ls[i] = exoscale.SecurityGroup{
			ID: exoscale.UUID(id.(string)),
		}
	}
	return
}

func ElasticIPsToElasticIPIDs(eips []exoscale.ElasticIP) (ls []string) {
	ls = make([]string, len(eips))
	for i, aag := range eips {
		ls[i] = aag.ID.String()
	}
	return
}

func ElasticIPIDsToElasticIPs(ids []any) (ls []exoscale.ElasticIP) {
	ls = make([]exoscale.ElasticIP, len(ids))
	for i, id := range ids {
		ls[i] = exoscale.ElasticIP{
			ID: exoscale.UUID(id.(string)),
		}
	}
	return
}

// OptionalString returns a null value for an empty string.
//
// The SDKv2 provider persisted unset optional strings as "", while the
// framework expects null. Normalising on read keeps state written before a
// resource was migrated to the framework from producing a permanent diff.
func OptionalString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}

	return types.StringValue(s)
}

// ImportStatePassthroughZonedID is a helper function for importing zoned resources.
// Works for all resources that can fully refresh with "id" and "zone" only.
// Input format is id@zone
func ImportStatePassthroughZonedID(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	idParts := strings.Split(req.ID, "@")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: id@zone. Got: %q", req.ID),
		)
		return
	}

	id, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}
	zone := idParts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx,
		path.Root("id"),
		types.StringValue(id.String()),
	)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(
		ctx,
		path.Root("zone"),
		types.StringValue(zone),
	)...)
}
