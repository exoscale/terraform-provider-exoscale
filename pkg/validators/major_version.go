package validators

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var majorVersionRe = regexp.MustCompile(`^\d+$`)

// IsMajorVersionValidator rejects version strings that contain a minor or
// patch component. Only plain integers (e.g. "2", "16") are accepted.
// This prevents "Provider produced inconsistent result after apply" errors
// caused by the API returning only the major version number.
var IsMajorVersionValidator validator.String = stringvalidator.RegexMatches(
	majorVersionRe,
	`version must be a major version number only (e.g. "2", not "2.0")`,
)
