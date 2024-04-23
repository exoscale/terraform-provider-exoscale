package block_storage

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VolumeSnapshotTargetModel defines nested data model.
type VolumeSnapshotTargetModel struct {
	ID types.String `tfsdk:"id"`
}

// Types returns nested data model types to be used for conversion.
func (m VolumeSnapshotTargetModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType,
	}
}

// VolumeInstanceModel defines nested data model.
type VolumeInstanceModel struct {
	ID types.String `tfsdk:"id"`
}

// Types returns nested data model types to be used for conversion.
func (m VolumeInstanceModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType,
	}
}

// VolumeSnapshotModel defines nested data model.
type VolumeSnapshotModel struct {
	ID types.String `tfsdk:"id"`
}

// Types returns nested data model types to be used for conversion.
func (m VolumeSnapshotModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType,
	}
}

// SnapshotVolumeModel defines nested data model.
type SnapshotVolumeModel struct {
	ID types.String `tfsdk:"id"`
}

// Types returns nested data model types to be used for conversion.
func (m SnapshotVolumeModel) Types() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType,
	}
}
