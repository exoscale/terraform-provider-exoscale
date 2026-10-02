package filter

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type matchStringFunc = func(given string) bool

func createMatchStringFunc(expected string) (matchStringFunc, error) {
	lenExp := len(expected)
	if lenExp > 1 && expected[0] == '/' && expected[lenExp-1] == '/' {
		r, err := regexp.Compile(expected[1 : lenExp-1])
		if err != nil {
			return nil, err
		}

		return func(given string) bool {
			return r.MatchString(given)
		}, nil
	}

	return func(given string) bool {
		return given == expected
	}, nil
}

type FilterFunc = func(map[string]any) bool

// NewEqualityFilter returns a filter matching the attribute argIdentifier against expected.
func NewEqualityFilter[T comparable](argIdentifier string, expected T) FilterFunc {
	return func(data map[string]any) bool {
		attr, ok := data[argIdentifier]
		if !ok {
			return false
		}

		switch v := attr.(type) {
		case *T:
			if *v == expected {
				return true
			}
		case T:
			if v == expected {
				return true
			}
		}

		return false
	}
}

func createStringFilterFunc(filterAttribute string, match matchStringFunc) FilterFunc {
	return func(data map[string]any) bool {
		attr, ok := data[filterAttribute]
		if !ok {
			return false
		}

		switch v := attr.(type) {
		case string:
			if match(v) {
				return true
			}
		case *string:
			if v != nil && match(*v) {
				return true
			}
		}

		return false
	}
}

// NewMapFilter returns a filter matching the map[string]string attribute argIdentifier:
// keys are matched exactly, while values may be matched as a regex if they begin and end with "/".
func NewMapFilter(ctx context.Context, argIdentifier string, expected map[string]string) (FilterFunc, error) {
	filters := make(map[string]matchStringFunc)
	for k, v := range expected {
		filter, err := createMatchStringFunc(v)
		if err != nil {
			return nil, err
		}

		filters[k] = filter
	}

	return func(data map[string]any) bool {
		mapAttr, ok := data[argIdentifier]
		if !ok {
			return false
		}

		mapToFilter, isMap := mapAttr.(map[string]string)
		if !isMap {
			tflog.Info(ctx, fmt.Sprintf("attribute %q has unexpected type %T", argIdentifier, mapAttr))

			return false
		}

		for filterKey, filterValue := range filters {
			value, ok := mapToFilter[filterKey]
			if !ok || !filterValue(value) {
				return false
			}
		}

		return true
	}, nil
}

// NewStringFilter returns a filter matching the string attribute argIdentifier against expected.
// If expected begins and ends with a "/" it is matched as a regex.
func NewStringFilter(argIdentifier, expected string) (FilterFunc, error) {
	matchFn, err := createMatchStringFunc(expected)
	if err != nil {
		return nil, err
	}

	return createStringFilterFunc(argIdentifier, matchFn), nil
}

// CheckForMatch returns true if all filters match on the given data.
func CheckForMatch(data map[string]any, filters []FilterFunc) bool {
	for _, filter := range filters {
		if !filter(data) {
			return false
		}
	}

	return true
}
