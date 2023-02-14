package exoscale

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type resourceDataGetter struct {
	d    *schema.ResourceData
	path string
}

func newResourceDataGetter(d *schema.ResourceData) resourceDataGetter {
	return resourceDataGetter{d: d}
}

func (rdg resourceDataGetter) Under(path string) resourceDataGetter {
	if rdg.path != "" && path != "" {
		rdg.path += "."
	}
	rdg.path += path

	return rdg
}

func (rdg resourceDataGetter) Get(path string) interface{} {
	v, ok := rdg.d.GetOk(rdg.Under(path).path)
	if ok {
		return v
	}

	return nil
}

func (rdg resourceDataGetter) GetInt64Ptr(path string) *int64 {
	v := rdg.Get(path)
	if v == nil {
		return nil
	}

	var r int64
	if i, ok := v.(int); ok {
		r = int64(i)
	} else {
		r = v.(int64)
	}

	return &r
}

func (rdg resourceDataGetter) GetBoolPtr(path string) *bool {
	// can't use GetOK here because it would false, false
	// if the value is set to false
	v := rdg.d.Get(rdg.Under(path).path).(bool)

	return &v
}

func (rdg resourceDataGetter) GetFloat64Ptr(path string) *float64 {
	v := rdg.Get(path)
	if v == nil {
		return nil
	}

	r := v.(float64)
	return &r
}

func (rdg resourceDataGetter) GetStringPtr(path string) *string {
	v := rdg.Get(path)
	if v == nil {
		return nil
	}

	r := v.(string)
	return &r
}

func (rdg resourceDataGetter) GetList(path string) []resourceDataGetter {
	v := rdg.Get(path)
	if v == nil {
		return nil
	}

	list := v.([]interface{})

	r := make([]resourceDataGetter, 0, len(list))
	for i := range list {
		r = append(r, rdg.Under(path).Under(strconv.Itoa(i)))
	}

	return r
}

func (rdg resourceDataGetter) GetStringSlicePtr(path string) *[]string {
	v := rdg.Get(path)
	if v == nil {
		return nil
	}

	s := v.([]interface{})

	r := make([]string, 0, len(s))
	for _, v := range s {
		r = append(r, v.(string))
	}

	return &r
}

func (rdg resourceDataGetter) GetMapPtr(path string) *map[string]interface{} {
	v := rdg.Get(path)
	if v == nil {
		return nil
	}

	r := v.(map[string]interface{})
	return &r
}

func (rdg resourceDataGetter) GetSet(path string) *[]string {
	v := rdg.Get(path)
	if v == nil {
		return nil
	}

	if l := v.(*schema.Set).Len(); l > 0 {
		list := make([]string, l)
		for i, j := range v.(*schema.Set).List() {
			list[i] = j.(string)
		}
		return &list
	}

	return nil
}
