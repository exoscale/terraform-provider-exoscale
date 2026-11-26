package database

import "testing"

func TestParseKafkaACLImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id          string
		wantService string
		wantACLID   string
		wantZone    string
		wantErr     bool
	}{
		{id: "my-kafka/acl5ec2571aff0@ch-gva-2", wantService: "my-kafka", wantACLID: "acl5ec2571aff0", wantZone: "ch-gva-2"},
		{id: "my-kafka/default@ch-gva-2", wantService: "my-kafka", wantACLID: "default", wantZone: "ch-gva-2"},
		{id: "my-kafka/acl5ec2571aff0", wantErr: true},
		{id: "my-kafka@ch-gva-2", wantErr: true},
		{id: "/acl5ec2571aff0@ch-gva-2", wantErr: true},
		{id: "my-kafka/@ch-gva-2", wantErr: true},
		{id: "my-kafka/acl/extra@ch-gva-2", wantErr: true},
		{id: "my-kafka/acl5ec2571aff0@", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()

			service, aclID, zone, err := parseKafkaACLImportID(tt.id)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got service=%q acl_id=%q zone=%q", service, aclID, zone)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if service != tt.wantService || aclID != tt.wantACLID || zone != tt.wantZone {
				t.Fatalf("got %q/%q@%q, want %q/%q@%q", service, aclID, zone, tt.wantService, tt.wantACLID, tt.wantZone)
			}
		})
	}
}
