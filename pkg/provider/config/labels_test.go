package config

import "testing"

func TestMergeLabels(t *testing.T) {
	t.Parallel()

	merged := MergeLabels(
		map[string]string{"env": "prod", "owner": "platform"},
		map[string]string{"owner": "team-a", "app": "web"},
	)

	if got, want := merged["env"], "prod"; got != want {
		t.Fatalf("env: got %q, want %q", got, want)
	}
	if got, want := merged["owner"], "team-a"; got != want {
		t.Fatalf("owner: got %q, want %q", got, want)
	}
	if got, want := merged["app"], "web"; got != want {
		t.Fatalf("app: got %q, want %q", got, want)
	}
}

func TestStripDefaultLabels(t *testing.T) {
	t.Parallel()

	labels := StripDefaultLabels(
		map[string]string{"env": "prod", "owner": "team-a", "app": "web"},
		map[string]string{"env": "prod", "owner": "platform"},
	)

	if _, ok := labels["env"]; ok {
		t.Fatal("expected env to be stripped")
	}
	if got, want := labels["owner"], "team-a"; got != want {
		t.Fatalf("owner: got %q, want %q", got, want)
	}
	if got, want := labels["app"], "web"; got != want {
		t.Fatalf("app: got %q, want %q", got, want)
	}
}
