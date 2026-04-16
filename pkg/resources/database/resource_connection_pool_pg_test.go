package database

import (
	"testing"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPGConnectionPoolResourceModelConfigurePostApplyRefreshFromConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		data               PGConnectionPoolResourceModel
		usernameConfigured bool
		modeConfigured     bool
		sizeConfigured     bool
		wantUsername       bool
		wantMode           bool
		wantSize           bool
	}{
		{
			name: "refresh omitted unknown fields on create",
			data: PGConnectionPoolResourceModel{
				Username: types.StringUnknown(),
				Mode:     types.StringUnknown(),
				Size:     types.Int64Unknown(),
			},
			wantUsername: true,
			wantMode:     true,
			wantSize:     true,
		},
		{
			name: "preserve configured optional fields",
			data: PGConnectionPoolResourceModel{
				Username: types.StringUnknown(),
				Mode:     types.StringUnknown(),
				Size:     types.Int64Unknown(),
			},
			usernameConfigured: true,
			modeConfigured:     true,
			sizeConfigured:     true,
		},
		{
			name: "preserve prior state when optional fields are omitted on update",
			data: PGConnectionPoolResourceModel{
				Username: types.StringValue("existing-user"),
				Mode:     types.StringValue("session"),
				Size:     types.Int64Value(10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := tt.data
			data.configurePostApplyRefreshFromConfig(tt.usernameConfigured, tt.modeConfigured, tt.sizeConfigured)

			if data.refreshUsername != tt.wantUsername {
				t.Fatalf("refreshUsername: got %v want %v", data.refreshUsername, tt.wantUsername)
			}
			if data.refreshMode != tt.wantMode {
				t.Fatalf("refreshMode: got %v want %v", data.refreshMode, tt.wantMode)
			}
			if data.refreshSize != tt.wantSize {
				t.Fatalf("refreshSize: got %v want %v", data.refreshSize, tt.wantSize)
			}
		})
	}
}

func TestPGConnectionPoolResourceModelApplyPostApplyConnectionPoolState(t *testing.T) {
	t.Parallel()

	pool := v3.DBAASServicePGConnectionPools{
		ConnectionURI: "postgres://pool",
		Database:      v3.DBAASDatabaseName("remote-db"),
		Mode:          v3.EnumPGPoolModeTransaction,
		Name:          v3.DBAASPGPoolName("pool"),
		Size:          v3.DBAASPGPoolSize(42),
		Username:      v3.DBAASPGPoolUsername("remote-user"),
	}

	t.Run("refreshes only computed and permitted optional fields", func(t *testing.T) {
		t.Parallel()

		data := PGConnectionPoolResourceModel{
			DatabaseName:    types.StringValue("configured-db"),
			Username:        types.StringValue("configured-user"),
			Mode:            types.StringValue("session"),
			Size:            types.Int64Value(10),
			ConnectionURI:   types.StringValue("postgres://stale"),
			refreshUsername: false,
			refreshMode:     false,
			refreshSize:     true,
		}

		data.applyPostApplyConnectionPoolState(pool)

		if got := data.DatabaseName.ValueString(); got != "configured-db" {
			t.Fatalf("database_name changed unexpectedly: got %q", got)
		}
		if got := data.Username.ValueString(); got != "configured-user" {
			t.Fatalf("username changed unexpectedly: got %q", got)
		}
		if got := data.Mode.ValueString(); got != "session" {
			t.Fatalf("mode changed unexpectedly: got %q", got)
		}
		if got := data.Size.ValueInt64(); got != 42 {
			t.Fatalf("size: got %d want %d", got, 42)
		}
		if got := data.ConnectionURI.ValueString(); got != pool.ConnectionURI {
			t.Fatalf("connection_uri: got %q want %q", got, pool.ConnectionURI)
		}
	})

	t.Run("hydrates omitted optional fields", func(t *testing.T) {
		t.Parallel()

		data := PGConnectionPoolResourceModel{
			Username:        types.StringUnknown(),
			Mode:            types.StringUnknown(),
			Size:            types.Int64Unknown(),
			refreshUsername: true,
			refreshMode:     true,
			refreshSize:     true,
		}

		data.applyPostApplyConnectionPoolState(pool)

		if got := data.Username.ValueString(); got != "remote-user" {
			t.Fatalf("username: got %q want %q", got, "remote-user")
		}
		if got := data.Mode.ValueString(); got != "transaction" {
			t.Fatalf("mode: got %q want %q", got, "transaction")
		}
		if got := data.Size.ValueInt64(); got != 42 {
			t.Fatalf("size: got %d want %d", got, 42)
		}
		if got := data.ConnectionURI.ValueString(); got != pool.ConnectionURI {
			t.Fatalf("connection_uri: got %q want %q", got, pool.ConnectionURI)
		}
	})
}
