package vkcs

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// BuildRequest takes an opts struct and builds a request body for
// Gophercloud to execute.
func BuildRequest(opts interface{}, parent string) (map[string]interface{}, error) {
	b, err := gophercloud.BuildRequestBody(opts, "")
	if err != nil {
		return nil, err
	}

	b = AddValueSpecs(b)

	return map[string]interface{}{parent: b}, nil
}

// checkDeleted checks the error to see if it's a 404 (Not Found) and, if so,
// sets the resource ID to the empty string instead of throwing an error.
func checkDeleted(d *schema.ResourceData, err error, msg string) error {
	if _, ok := err.(gophercloud.ErrDefault404); ok {
		d.SetId("")
		return nil
	}

	return fmt.Errorf("%s %s: %s", msg, d.Id(), err)
}

// getRegion returns the region that was specified in the resource. If a
// region was not set, the provider-level region is checked. The provider-level
// region can either be set by the region argument or by OS_REGION_NAME.
func getRegion(d *schema.ResourceData, config configer) string {
	if v, ok := d.GetOk("region"); ok {
		return v.(string)
	}

	return config.GetRegion()
}

// AddValueSpecs expands the 'value_specs' object and removes 'value_specs'
// from the reqeust body.
func AddValueSpecs(body map[string]interface{}) map[string]interface{} {
	if body["value_specs"] != nil {
		for k, v := range body["value_specs"].(map[string]interface{}) {
			body[k] = v
		}
		delete(body, "value_specs")
	}

	return body
}

// MapValueSpecs converts ResourceData into a map.
func MapValueSpecs(d *schema.ResourceData) map[string]string {
	m := make(map[string]string)
	for key, val := range d.Get("value_specs").(map[string]interface{}) {
		m[key] = val.(string)
	}
	return m
}

func checkForRetryableError(err error) *resource.RetryError {
	switch e := err.(type) {
	case gophercloud.ErrDefault500:
		return resource.RetryableError(err)
	case gophercloud.ErrDefault409:
		return resource.RetryableError(err)
	case gophercloud.ErrDefault503:
		return resource.RetryableError(err)
	case gophercloud.ErrUnexpectedResponseCode:
		if e.GetStatusCode() == 504 || e.GetStatusCode() == 502 {
			return resource.RetryableError(err)
		} else {
			return resource.NonRetryableError(err)
		}
	default:
		return resource.NonRetryableError(err)
	}
}

func expandVendorOptions(vendOptsRaw []interface{}) map[string]interface{} {
	vendorOptions := make(map[string]interface{})

	for _, option := range vendOptsRaw {
		for optKey, optValue := range option.(map[string]interface{}) {
			vendorOptions[optKey] = optValue
		}
	}

	return vendorOptions
}

func expandObjectReadTags(d *schema.ResourceData, tags []string) {
	d.Set("all_tags", tags)

	allTags := d.Get("all_tags").(*schema.Set)
	desiredTags := d.Get("tags").(*schema.Set)
	actualTags := allTags.Intersection(desiredTags)
	if !actualTags.Equal(desiredTags) {
		d.Set("tags", expandToStringSlice(actualTags.List()))
	}
}

func expandObjectUpdateTags(d *schema.ResourceData) []string {
	allTags := d.Get("all_tags").(*schema.Set)
	oldTagsRaw, newTagsRaw := d.GetChange("tags")
	oldTags, newTags := oldTagsRaw.(*schema.Set), newTagsRaw.(*schema.Set)

	allTagsWithoutOld := allTags.Difference(oldTags)

	return expandToStringSlice(allTagsWithoutOld.Union(newTags).List())
}

func expandObjectTags(d *schema.ResourceData) []string {
	rawTags := d.Get("tags").(*schema.Set).List()
	tags := make([]string, len(rawTags))

	for i, raw := range rawTags {
		tags[i] = raw.(string)
	}

	return tags
}

func expandToMapStringString(v map[string]interface{}) map[string]string {
	m := make(map[string]string)
	for key, val := range v {
		if strVal, ok := val.(string); ok {
			m[key] = strVal
		}
	}

	return m
}

func expandToStringSlice(v []interface{}) []string {
	s := make([]string, len(v))
	for i, val := range v {
		if strVal, ok := val.(string); ok {
			s[i] = strVal
		}
	}

	return s
}

// strSliceContains checks if a given string is contained in a slice
// When anybody asks why Go needs generics, here you go.
func strSliceContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func sliceUnion(a, b []string) []string {
	var res []string
	for _, i := range a {
		if !strSliceContains(res, i) {
			res = append(res, i)
		}
	}
	for _, k := range b {
		if !strSliceContains(res, k) {
			res = append(res, k)
		}
	}
	return res
}
