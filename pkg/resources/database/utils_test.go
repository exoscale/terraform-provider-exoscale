package database_test

import (
	"reflect"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/database"
)

func TestPartialSettingsPatch(t *testing.T) {
	t.Parallel()

	type testCaseInput struct {
		data  map[string]any
		patch map[string]any
	}
	type testCase struct {
		input  testCaseInput
		result map[string]any
	}

	cases := []testCase{
		{
			input: testCaseInput{
				data: map[string]any{
					"key": "value",
				},
				patch: map[string]any{
					"key": "newvalue",
				},
			},
			result: map[string]any{
				"key": "newvalue",
			},
		},
		{
			input: testCaseInput{
				data: map[string]any{
					"key": "value",
				},
				patch: map[string]any{
					"ke2": "newvalu2",
				},
			},
			result: map[string]any{},
		},
		{
			input: testCaseInput{
				data: map[string]any{},
				patch: map[string]any{
					"key": "value",
				},
			},
			result: map[string]any{},
		},
		{
			input: testCaseInput{
				data: map[string]any{
					"key1": "value",
					"key2": 1,
				},
				patch: map[string]any{
					"key2": 2,
					"key3": "newvalue",
				},
			},
			result: map[string]any{
				"key2": 2,
			},
		},
	}

	for _, c := range cases {
		database.PartialSettingsPatch(c.input.data, c.input.patch)

		if !reflect.DeepEqual(c.input.data, c.result) {
			t.Fatalf("not equal: %v %v", c.input.data, c.result)
		}
	}
}
