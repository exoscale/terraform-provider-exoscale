package database

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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
		{
			// Nested settings: only the sub-keys the user set are managed;
			// server-injected defaults inside the nested object are dropped.
			input: testCaseInput{
				data: map[string]any{
					"server_settings": map[string]any{
						"vector_similarity_index_cache_size": 0.1,
					},
					"tiered_storage_move_factor": 0.3,
				},
				patch: map[string]any{
					"server_settings": map[string]any{
						"vector_similarity_index_cache_size": 0.1,
						"max_memory_usage":                   1024,
					},
					"tiered_storage_move_factor": 0.3,
				},
			},
			result: map[string]any{
				"server_settings": map[string]any{
					"vector_similarity_index_cache_size": 0.1,
				},
				"tiered_storage_move_factor": 0.3,
			},
		},
		{
			// CLI drift: a value the user set is changed out-of-band (CLI /
			// console). The live (patched) value is picked up into data so
			// the plan shows the drift, while server-injected defaults are
			// still not added.
			input: testCaseInput{
				data: map[string]any{
					"server_settings": map[string]any{
						"vector_similarity_index_cache_size": 0.1,
					},
					"tiered_storage_move_factor": 0.3,
				},
				patch: map[string]any{
					"server_settings": map[string]any{
						"vector_similarity_index_cache_size": 0.5,
						"max_memory_usage":                   1024,
					},
					"tiered_storage_move_factor": 0.2,
				},
			},
			result: map[string]any{
				"server_settings": map[string]any{
					"vector_similarity_index_cache_size": 0.5,
				},
				"tiered_storage_move_factor": 0.2,
			},
		},
	}

	for _, c := range cases {
		PartialSettingsPatch(c.input.data, c.input.patch)

		if !reflect.DeepEqual(c.input.data, c.result) {
			t.Fatalf("not equal: %v %v", c.input.data, c.result)
		}
	}
}

func Test_uriWitoutCreds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		uri    *string
		output *string
		err    string
	}{
		{
			name:   "nominal with creds",
			uri:    new("postgres://user:password@my-dbaas.k.aivencloud.com:21699/defaultdb?sslmode=require"),
			output: new("postgres://my-dbaas.k.aivencloud.com:21699/defaultdb?sslmode=require"),
		},
		{
			name:   "nominal without creds",
			uri:    new("postgres://my-dbaas.k.aivencloud.com:21699/defaultdb?sslmode=require"),
			output: new("postgres://my-dbaas.k.aivencloud.com:21699/defaultdb?sslmode=require"),
		},
		{
			name: "nominal no value",
		},
	}

	for _, ut := range tests {
		t.Run(ut.name, func(t *testing.T) {
			output, err := uriWitoutCreds(ut.uri)

			assert.Equal(t, ut.output, output)
			if ut.err != "" {
				assert.ErrorContains(t, err, ut.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
