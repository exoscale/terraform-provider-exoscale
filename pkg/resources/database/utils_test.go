package database_test

import (
	"reflect"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/database"
)

func TestPartialSettingsPatch(t *testing.T) {
	type testCaseInput struct {
		data  map[string]interface{}
		patch map[string]interface{}
	}
	type testCase struct {
		input  testCaseInput
		result map[string]interface{}
	}

	cases := []testCase{
		{
			input: testCaseInput{
				data: map[string]interface{}{
					"key": "value",
				},
				patch: map[string]interface{}{
					"key": "newvalue",
				},
			},
			result: map[string]interface{}{
				"key": "newvalue",
			},
		},
		{
			input: testCaseInput{
				data: map[string]interface{}{
					"key": "value",
				},
				patch: map[string]interface{}{
					"ke2": "newvalu2",
				},
			},
			result: map[string]interface{}{},
		},
		{
			input: testCaseInput{
				data: map[string]interface{}{},
				patch: map[string]interface{}{
					"key": "value",
				},
			},
			result: map[string]interface{}{},
		},
		{
			input: testCaseInput{
				data: map[string]interface{}{
					"key1": "value",
					"key2": 1,
				},
				patch: map[string]interface{}{
					"key2": 2,
					"key3": "newvalue",
				},
			},
			result: map[string]interface{}{
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
