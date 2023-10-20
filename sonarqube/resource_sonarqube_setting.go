package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Setting struct {
	Key         string              `json:"key"`
	Value       string              `json:"value"`
	Values      []string            `json:"values"`
	Inherited   bool                `json:"inherited"`
	FieldValues []map[string]string `json:"fieldValues"`
}

type GetSettings struct {
	Setting            []Setting `json:"settings"`
	SetSecuredSettings []string  `json:"setSecuredSettings"`
}

func (a Setting) ToMap() map[string]interface{} {
	obj := make(map[string]interface{})

	obj["value"] = a.Value
	if a.Values != nil {
		obj["values"] = a.Values
	}
	if a.FieldValues != nil {
		obj["fieldValues"] = a.FieldValues
	}
	return obj
}

func resourceSonarqubeSettings() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeSettingsCreate,
		Read:   resourceSonarqubeSettingsRead,
		Update: resourceSonarqubeSettingsUpdate,
		Delete: resourceSonarqubeSettingsDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeSettingsImporter,
		},

		Schema: map[string]*schema.Schema{
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Setting key",
			},
			"value": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Setting value. To reset a value, please use the reset web service.",
				ExactlyOneOf: []string{"value", "values", "field_values"},
			},
			"values": {
				Type:         schema.TypeList,
				Optional:     true,
				Description:  "Setting multi values for the supplied key",
				ExactlyOneOf: []string{"value", "values", "field_values"},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"field_values": {
				Type:         schema.TypeList,
				Optional:     true,
				Description:  "Setting field values for the supplied key",
				ExactlyOneOf: []string{"value", "values", "field_values"},
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: schema.TypeString,
				},
			},
		},
	}
}

func resourceSonarqubeSettingsCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/settings/set"
	sonarQubeURL.RawQuery = getCreateOrUpdateQueryRawQuery([]string{d.Get("key").(string)}, d)

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeSettingsCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d.SetId(d.Get("key").(string))
	return resourceSonarqubeSettingsRead(d, m)
}

func resourceSonarqubeSettingsRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/settings/values"
	sonarQubeURL.RawQuery = url.Values{
		"keys": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeSettingsRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	settingReadResponse := GetSettings{}
	err = json.NewDecoder(resp.Body).Decode(&settingReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeSettingsRead: Failed to decode json into struct: %+v", err)
	}

	for _, value := range settingReadResponse.Setting {
		if d.Id() == value.Key {
			d.Set("key", value.Key)
			d.Set("value", value.Value)
			d.Set("values", value.Values)
			d.Set("field_values", value.FieldValues)
			d.SetId(value.Key)
			return nil
		}
	}
	return fmt.Errorf("resourceSonarqubeSettingsRead: Failed to find setting: %+v", d.Id())
}

func resourceSonarqubeSettingsDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/settings/reset"
	sonarQubeURL.RawQuery = url.Values{
		"keys": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeSettingsDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeSettingsImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	d.Set("key", d.Id())
	if err := resourceSonarqubeSettingsRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func resourceSonarqubeSettingsUpdate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/settings/set"

	sonarQubeURL.RawQuery = getCreateOrUpdateQueryRawQuery([]string{d.Id()}, d)

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeSettingsUpdate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return resourceSonarqubeSettingsRead(d, m)
}
func getCreateOrUpdateQueryRawQuery(key []string, d *schema.ResourceData) string {
	// build the base query
	RawQuery := url.Values{
		"key": key,
	}
	// Add in value/values/fieldValues as appropriate
	// single value
	if value, ok := d.GetOk("value"); ok {
		RawQuery.Add("value", value.(string))
	} else {
		// array of strings
		if values, ok := d.GetOk("values"); ok {
			for _, value := range values.([]interface{}) {
				RawQuery.Add("values", value.(string))
			}
		} else {
			// array of objects of key/value pairs
			fieldValues := d.Get("field_values").([]interface{})
			for _, value := range fieldValues {
				b, _ := json.Marshal(value)
				fv := string(b)
				RawQuery.Add("fieldValues", fv)
			}
		}
	}
	return RawQuery.Encode()
}

/* This content is used for settings parameter in multiple resources ('project', 'portfolio')  */
func getComponentSettings(component string, m interface{}) ([]Setting, error) {
	if component == "" {
		return []Setting{}, nil
	}
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/settings/values"
	sonarQubeURL.RawQuery = url.Values{"component": []string{component}}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"getProjectSettings",
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	settingReadResponse := GetSettings{}
	err = json.NewDecoder(resp.Body).Decode(&settingReadResponse)
	if err != nil {
		return nil, fmt.Errorf("getProjectSettings: Failed to decode json into struct: %+v", err)
	}

	settingsList := make([]Setting, 0)
	// Filter settings (removing inherited)
	for _, e := range settingReadResponse.Setting {
		if e.Inherited == false {
			settingsList = append(settingsList, e)
		}
	}

	// Make sure the order is always the same for when we are comparing lists of conditions
	sort.Slice(settingsList, func(i, j int) bool {
		return settingsList[i].Key < settingsList[j].Key
	})

	return settingsList, nil
}

func synchronizeSettings(d *schema.ResourceData, m interface{}) (bool, error) {
	changed := false
	componentId := d.Id()
	componentSettings := d.Get("setting").([]interface{})

	apiComponentSettings, _ := getComponentSettings(componentId, m)

	// Make sure the order is always the same for when we are comparing lists of conditions
	sort.Slice(componentSettings, func(i, j int) bool {
		return componentSettings[i].(map[string]interface{})["key"].(string) < componentSettings[j].(map[string]interface{})["key"].(string)
	})

	// Determine which conditions have been added or changed and update those
	for _, s := range componentSettings {
		setting := s.(map[string]interface{})
		key := setting["key"].(string)

		// Update the condition if it already exists
		for _, apiSetting := range apiComponentSettings {
			if key == apiSetting.Key {
				if checkSettingDiff(setting, apiSetting) {
					err := setComponentSetting(componentId, setting, m, &changed)
					if err != nil {
						return false, fmt.Errorf("addOrUpdateCondition: Failed to update setting '%s': %+v", key, err)
					}
				}
			}
		}
	}

	// Determine if any settings have been removed and delete them
	err := removeComponentSettings(componentId, componentSettings, &apiComponentSettings, m, &changed)
	if err != nil {
		return changed, err
	}

	if changed {
		d.Set("setting", componentSettings)
	}

	return changed, nil
}

func checkSettingDiff(a map[string]interface{}, b Setting) bool {
	if a["value"] != nil {
		return a["value"].(string) != b.Value
	} else if a["values"] != nil {
		// array of strings
		values := a["field_values"].([]string)
		if len(values) != len(b.FieldValues) {
			return false
		}
		for i := range values {
			if string(values[i]) != string(b.Values[i]) {
				return false
			}
		}
		return true
	} else if a["field_values"] != nil {
		// array of objects of key/value pairs
		fieldValues := a["field_values"].([]interface{})
		if len(fieldValues) != len(b.FieldValues) {
			return false
		}
		for i := range fieldValues {
			k1, _ := json.Marshal(fieldValues[i])
			k2, _ := json.Marshal(b.FieldValues[i])
			if string(k1) != string(k2) {
				return false
			}
		}
		return true
	}
	return false
}

func getComponentSettingUrlEncode(setting map[string]interface{}) url.Values {
	raw := url.Values{
		"key": []string{setting["key"].(string)},
	}
	if setting["value"] != nil {
		raw.Add("value", setting["value"].(string))
	} else if setting["values"] != nil {
		// array of strings
		for _, value := range setting["values"].([]interface{}) {
			raw.Add("values", value.(string))
		}
	} else if setting["field_values"] != nil {
		// array of objects of key/value pairs
		fieldValues := setting["field_values"].([]interface{})
		for _, value := range fieldValues {
			b, _ := json.Marshal(value)
			fv := string(b)
			raw.Add("fieldValues", fv)
		}
	}
	return raw
}

func setComponentSetting(component string, setting map[string]interface{}, m interface{}, changed *bool) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/settings/set"
	params := getComponentSettingUrlEncode(setting)
	params.Add("component", component)
	sonarQubeURL.RawQuery = params.Encode()

	_, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"setComponentSettings",
	)
	if err != nil {
		return fmt.Errorf("setComponentSettings: Failed to set project setting key=%s: %+v", setting["key"].(string), err)
	}
	*changed = true

	return nil
}

func removeComponentSettings(component string, newSettings []interface{}, apiProjectSettings *[]Setting, m interface{}, changed *bool) error {
	if component == "" {
		return nil
	}

	var toDelete []string
	for _, apiSetting := range *apiProjectSettings {
		found := false
		for _, newSetting := range newSettings {
			newSetting_ := newSetting.(map[string]interface{})
			if newSetting_["key"].(string) == apiSetting.Key {
				found = true
				break
			}
		}
		if !found {
			toDelete = append(toDelete, fmt.Sprint(apiSetting.Key))
		}
	}
	// Delete not found
	if len(toDelete) > 0 {
		sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/settings/reset"
		sonarQubeURL.RawQuery = url.Values{
			"component": []string{component},
			"keys":      []string{strings.Join(toDelete, ",")},
		}.Encode()

		_, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL,
			http.StatusNoContent,
			"deleteSetting",
		)

		if err != nil {
			return fmt.Errorf("removeComponentSettings: Failed to delete setting %s: %+v", component, err)
		}
		*changed = true
	}
	return nil
}
