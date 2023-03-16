package filter

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

type FilterFunc = func(map[string]interface{}) bool

func createEqualityFilter[T comparable](argIdentifier string, expected T) (FilterFunc, error) {
	return func(data map[string]interface{}) bool {
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
	}, nil
}

func createStringFilterFunc(filterAttribute string, match matchStringFunc) FilterFunc {
	return func(data map[string]interface{}) bool {
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
			if match(*v) {
				return true
			}
		}

		return false
	}
}

func createMapStrToStrFilterFunc(ctx context.Context, argIdentifier string, filterProp interface{}) (FilterFunc, error) {
	filters := make(map[string]matchStringFunc)
	maps := filterProp.(map[string]interface{})
	for k, v := range maps {
		filter, err := createMatchStringFunc(v.(string))
		if err != nil {
			return nil, err
		}

		filters[k] = filter
	}

	return func(data map[string]interface{}) bool {
		mapAttr, ok := data[argIdentifier]
		if !ok {
			return false
		}

		mapToFilter, isMap := mapAttr.(map[string]string)
		if !isMap {
			tflog.Info(ctx, fmt.Sprintf("attribute of compute instance has unexpected type %T for labels", mapAttr))

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

func createStringFilter(argIdentifier, expected string) (FilterFunc, error) {
	matchFn, err := createMatchStringFunc(expected)
	if err != nil {
		return nil, err
	}

	return createStringFilterFunc(argIdentifier, matchFn), nil
}

// CreateFilters accepts a schema for a data source and creates a filter.FilterFunc for each attribute of type bool, int, string or map[string]string. Use these filters to create aggregate data sources like lists.
func CreateFilters(ctx context.Context, d *schema.ResourceData, s map[string]*schema.Schema) ([]FilterFunc, error) {
	var filters []FilterFunc

	for argIdentifier, argSpec := range s {
		argValue, ok := d.GetOk(argIdentifier)
		if !ok {
			continue
		}

		switch argSpec.Type {
		case schema.TypeBool:
			newFilterFunc, err := createEqualityFilter(argIdentifier, argValue.(bool))
			if err != nil {
				return nil, err
			}

			filters = append(filters, newFilterFunc)
		case schema.TypeInt:
			newFilterFunc, err := createEqualityFilter(argIdentifier, int64(argValue.(int)))
			if err != nil {
				return nil, err
			}

			filters = append(filters, newFilterFunc)
		case schema.TypeString:
			newFilterFunc, err := createStringFilter(argIdentifier, argValue.(string))
			if err != nil {
				return nil, err
			}

			filters = append(filters, newFilterFunc)
		case schema.TypeMap:
			newFilter, err := createMapStrToStrFilterFunc(ctx, argIdentifier, argValue)
			if err != nil {
				return nil, err
			}

			filters = append(filters, newFilter)
		default:
			continue
		}
	}

	return filters, nil
}

// CheckForMatch returns true if all filters match on the given data.
func CheckForMatch(data map[string]interface{}, filters []FilterFunc) bool {
	for _, filter := range filters {
		if !filter(data) {
			return false
		}
	}

	return true
}

func createFilterAttribute(typ schema.ValueType) *schema.Schema {
	filterMessage := ""
	switch typ {
	case schema.TypeBool:
		filterMessage = "Match against this bool"
	case schema.TypeInt:
		filterMessage = "Match against this int"
	case schema.TypeString:
		filterMessage = "Match against this string. If you supply a string that begins and ends with a \"/\" it will be matched as a regex."
	}

	return &schema.Schema{
		Description: filterMessage,
		Type:        typ,
		Optional:    true,
	}
}

func createMapFilterAttribute() *schema.Schema {
	return &schema.Schema{
		Description: "Match against key/values. Keys are matched exactly, while values may be matched as a regex if you supply a string that begins and ends with \"/\"",
		Type:        schema.TypeMap,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Optional:    true,
	}
}

// AddFilterAttributes adds filter attributes to your resource for all bool, int, string and map[string]string attributes in the supplied schema. In combination with CreateFilters you may use this to create aggregate data sources with filtering functionality.
func AddFilterAttributes(r *schema.Resource, s map[string]*schema.Schema) {
	for attrIdentifier, attrSpec := range s {
		switch attrSpec.Type {
		case schema.TypeBool, schema.TypeInt, schema.TypeString:
			r.Schema[attrIdentifier] = createFilterAttribute(attrSpec.Type)
		case schema.TypeMap:
			elem, ok := attrSpec.Elem.(*schema.Schema)
			if ok && elem.Type == schema.TypeString {
				r.Schema[attrIdentifier] = createMapFilterAttribute()
			}
		}
	}
}
