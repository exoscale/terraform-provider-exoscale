package filter

import (
	"context"
	"testing"
)

func TestComputeInstanceListFilterString(t *testing.T) {
	attributeToMatch := "my-test-attr"
	valueToMatch := "string-to-match"

	dataToFilter := map[string]interface{}{
		attributeToMatch: valueToMatch,
	}

	matchFn, err := createMatchStringFunc(valueToMatch)
	if err != nil {
		t.Fatal(err)
	}

	filter := createStringFilterFunc(attributeToMatch, matchFn)

	if !CheckForMatch(dataToFilter, []FilterFunc{filter}) {
		t.Error("should match")
	}
}

func TestComputeInstanceListFilterRegex(t *testing.T) {
	attributeToMatch := "my-test-attr"
	valueToMatch := "string-123-to-match-by-regex"

	dataToFilter := map[string]interface{}{
		attributeToMatch: valueToMatch,
	}

	matchFn, err := createMatchStringFunc("/.*123.*/")
	if err != nil {
		t.Fatal(err)
	}

	filter := createStringFilterFunc(attributeToMatch, matchFn)

	if !CheckForMatch(dataToFilter, []FilterFunc{filter}) {
		t.Error("should match")
	}
}

func TestComputeInstanceListFilterLabelsExactly(t *testing.T) {
	labelToMatch := "my-label"

	dataToFilter := map[string]interface{}{
		"labels": map[string]string{
			labelToMatch: "label-string-to-match",
		},
	}

	labelsFilterProp := map[string]interface{}{
		labelToMatch: "label-string-to-match",
	}

	filter, err := createMapStrToStrFilterFunc(context.Background(), "labels", labelsFilterProp)
	if err != nil {
		t.Fatal(err)
	}

	if !CheckForMatch(dataToFilter, []FilterFunc{filter}) {
		t.Error("should match")
	}
}

func TestComputeInstanceListFilterLabelsRegex(t *testing.T) {
	labelToMatch := "my-label"

	dataToFilter := map[string]interface{}{
		"labels": map[string]string{
			labelToMatch: "label-string-to-match",
		},
	}

	labelsFilterProp := map[string]interface{}{
		labelToMatch: "/.*-to.*-/",
	}

	filter, err := createMapStrToStrFilterFunc(context.Background(), "labels", labelsFilterProp)
	if err != nil {
		t.Fatal(err)
	}

	if !CheckForMatch(dataToFilter, []FilterFunc{filter}) {
		t.Error("should match")
	}
}
