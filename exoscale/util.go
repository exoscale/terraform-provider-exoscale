package exoscale

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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

// defaultString returns the value of the string pointer v if not nil, otherwise the default value specified.
func defaultString(v *string, def string) string {
	if v != nil {
		return *v
	}

	return def
}

// defaultInt64 returns the value of the int64 pointer v if not nil, otherwise the default value specified.
func defaultInt64(v *int64, def int64) int64 {
	if v != nil {
		return *v
	}

	return def
}

// defaultBool returns the value of the bool pointer v if not nil, otherwise the default value specified.
func defaultBool(v *bool, def bool) bool {
	if v != nil {
		return *v
	}

	return def
}

// addressToStringPtr returns a string representation of addr if not nil, otherwise nil.
func addressToStringPtr(addr *net.IP) *string {
	if addr != nil {
		addrStr := addr.String()
		return &addrStr
	}

	return nil
}

// nonEmptyStringPtr returns a non-nil pointer to s if the string is not empty, otherwise nil.
func nonEmptyStringPtr(s string) *string {
	if s != "" {
		return &s
	}

	return nil
}

func schemaSetToStringArray(set *schema.Set) []string {
	array := make([]string, set.Len())
	for i, group := range set.List() {
		array[i] = group.(string)
	}

	return array
}

func unique(s []string) []string {
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

// DiffSuppressFunc https://www.terraform.io/plugin/sdkv2/schemas/schema-behaviors#diffsuppressfunc
// Do no show case differences between state and resource
func suppressCaseDiff(k, old, new string, d *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}

// user-data compression and base64 encoding, used in resource_exoscale_compute[_instance[_pool]]
// returns (user_data, user_data_already_base64, error)
func encodeUserData(userData string) (string, bool, error) {
	// template_cloudinit_config alows to gzip but not base64, prevent such case
	if len(userData) > 2 && userData[0] == '\x1f' && userData[1] == '\x8b' {
		return "", false, errors.New("user_data appears to be gzipped: it should be left raw, or also be base64 encoded")
	}

	// If user supplied data is already base64 encoded, do nothing.
	_, err := base64.StdEncoding.DecodeString(userData)
	if err == nil {
		return userData, true, nil
	}

	b := new(bytes.Buffer)
	gz := gzip.NewWriter(b)

	if _, err := gz.Write([]byte(userData)); err != nil {
		return "", false, err
	}
	if err := gz.Flush(); err != nil {
		return "", false, err
	}
	if err := gz.Close(); err != nil {
		return "", false, err
	}

	userDataBase64 := base64.StdEncoding.EncodeToString(b.Bytes())

	if len(userDataBase64) >= computeMaxUserDataLength {
		return "", false, fmt.Errorf("user-data maximum allowed length is %d bytes", computeMaxUserDataLength)
	}

	return userDataBase64, false, nil
}

// user-data base64 decoding & decompression, used in resource_exoscale_compute[_instance[_pool]]
func decodeUserData(data string) (string, error) {
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

	userData, err := ioutil.ReadAll(gz)
	if err != nil {
		return "", err
	}

	return string(userData), nil
}

func parseIAMAccessKeyResource(v string) (*egoscale.IAMAccessKeyResource, error) {
	var iamAccessKeyResource egoscale.IAMAccessKeyResource

	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format")
	}
	iamAccessKeyResource.ResourceName = parts[1]

	parts = strings.SplitN(parts[0], "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format")
	}
	iamAccessKeyResource.Domain = parts[0]
	iamAccessKeyResource.ResourceType = parts[1]

	if iamAccessKeyResource.Domain == "" ||
		iamAccessKeyResource.ResourceType == "" ||
		iamAccessKeyResource.ResourceName == "" {
		return nil, fmt.Errorf("invalid format")
	}

	return &iamAccessKeyResource, nil
}

var zones = []string{
	"ch-gva-2",
	"ch-dk-2",
	"at-vie-1",
	"de-fra-1",
	"bg-sof-1",
	"de-muc-1",
}

func validateZone() schema.SchemaValidateFunc {
	return validation.StringInSlice(zones, false)
}
