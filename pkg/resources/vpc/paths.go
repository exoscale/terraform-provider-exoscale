package vpc

import "github.com/hashicorp/terraform-plugin-framework/path"

// Shared attribute paths used by ImportState implementations across the
// vpc_subnet, vpc_route and vpc_subnet_attachment resources.
var (
	pathID         = path.Root("id")
	pathVpcID      = path.Root("vpc_id")
	pathSubnetID   = path.Root("subnet_id")
	pathZone       = path.Root("zone")
	pathInstanceID = path.Root("instance_id")
)
