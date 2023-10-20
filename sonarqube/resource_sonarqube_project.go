package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Project used in CreateProjectResponse
type Project struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Qualifier string `json:"qualifier"`
}

// GetProject for unmarshalling response body from getting project details
type GetProject struct {
	Component ProjectComponent `json:"component"`
}

// ProjectComponents used in GetProject
type ProjectComponent struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Qualifier    string   `json:"qualifier"`
	AnalysisDate string   `json:"analysisDate"`
	Version      string   `json:"version"`
	Tags         []string `json:"tags,omitempty"`
	Visibility   string   `json:"visibility"`
}

// CreateProjectResponse for unmarshalling response body of project creation
type CreateProjectResponse struct {
	Project Project `json:"project"`
}

// Returns the resource represented by this file.
func resourceSonarqubeProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeProjectCreate,
		Read:   resourceSonarqubeProjectRead,
		Update: resourceSonarqubeProjectUpdate,
		Delete: resourceSonarqubeProjectDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeProjectImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"project": {
				Type:     schema.TypeString,
				Required: true,
			},
			"visibility": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "public",
			},
			"tags": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"setting": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    false,
				Description: "A list of settings associated to the project",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Setting key",
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Setting a value for the supplied key",
						},
						"values": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Setting multi values for the supplied key",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"field_values": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Setting field values for the supplied key",
							Elem: &schema.Schema{
								Type: schema.TypeMap,
								Elem: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func projectSetTags(d *schema.ResourceData, m interface{}, sonarQubeURL url.URL) error {
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/project_tags/set"

	// TODO: Create a helper file for convertListToCSV or something. This is used in Portfolio too
	var tags []string
	for _, v := range d.Get("tags").([]interface{}) {
		tags = append(tags, fmt.Sprint(v))
	}
	tagsCSV := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(tags)), ","), "[]")
	sonarQubeURL.RawQuery = url.Values{
		"project": []string{d.Get("project").(string)},
		"tags":    []string{tagsCSV},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubePortfolioCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeProjectCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/projects/create"

	sonarQubeURL.RawQuery = url.Values{
		"name":       []string{d.Get("name").(string)},
		"project":    []string{d.Get("project").(string)},
		"visibility": []string{d.Get("visibility").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeProjectCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = projectSetTags(d, m, m.(*ProviderConfiguration).sonarQubeURL)
	if err != nil {
		return err
	}

	// Decode response into struct
	projectResponse := CreateProjectResponse{}
	err = json.NewDecoder(resp.Body).Decode(&projectResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeProjectCreate: Failed to decode json into struct: %+v", err)
	}

	d.SetId(projectResponse.Project.Key)

	// Set settings
	_, err = synchronizeSettings(d, m)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeProjectCreate: Failed to sync project settings: %+v", err)
	}

	return resourceSonarqubeProjectRead(d, m)
}

func resourceSonarqubeProjectRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/components/show"
	sonarQubeURL.RawQuery = url.Values{
		"component": []string{d.Get("project").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeProjectRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	projectReadResponse := GetProject{}

	err = json.NewDecoder(resp.Body).Decode(&projectReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeProjectRead: Failed to decode json into struct: %+v", err)
	}

	d.SetId(projectReadResponse.Component.Key)
	d.Set("name", projectReadResponse.Component.Name)
	d.Set("project", projectReadResponse.Component.Key)
	d.Set("visibility", projectReadResponse.Component.Visibility)

	// Get settings
	var projectSettings []Setting
	if _, ok := d.GetOk("setting"); ok {
		projectSettings, err = getComponentSettings(d.Id(), m)
		if err != nil {
			return fmt.Errorf("resourceSonarqubeProjectRead: Failed to read project settings: %+v", err)
		}

		if len(projectSettings) > 0 {
			settings := make([]interface{}, len(projectSettings))
			for i, s := range projectSettings {
				settings[i] = s.ToMap()
			}
			d.Set("setting", settings)
		}
	}

	if len(projectReadResponse.Component.Tags) > 0 {
		d.Set("tags", projectReadResponse.Component.Tags)
	}

	return nil
}

func resourceSonarqubeProjectUpdate(d *schema.ResourceData, m interface{}) error {

	// handle default updates (api/users/update)
	if d.HasChange("visibility") {
		sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/projects/update_visibility"
		sonarQubeURL.RawQuery = url.Values{
			"project":    []string{d.Get("project").(string)},
			"visibility": []string{d.Get("visibility").(string)},
		}.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL,
			http.StatusNoContent,
			"resourceSonarqubeProjectUpdate",
		)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube project: %+v", err)
		}
		defer resp.Body.Close()
	}

	if d.HasChanges("tags") {
		err := projectSetTags(d, m, m.(*ProviderConfiguration).sonarQubeURL)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube selection mode: %+v", err)
		}
	}

	// handle project key updates (api/projects/update_key)
	if d.HasChange("project") {
		oldKey, newKey := d.GetChange("project")

		sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/projects/update_key"
		sonarQubeURL.RawQuery = url.Values{
			"from": []string{oldKey.(string)},
			"to":   []string{newKey.(string)},
		}.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL,
			http.StatusNoContent,
			"resourceSonarqubeProjectUpdate",
		)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube project key: %+v", err)
		}
		defer resp.Body.Close()

		// Update the id like in github provider (https://github.com/integrations/terraform-provider-github/blob/b7e63d63c59b9b1df9c6d05204bdaa1b349e8c8a/github/resource_github_repository.go#L746-L750)
		d.SetId(newKey.(string))
	}

	if d.HasChange("setting") {
		_, err := synchronizeSettings(d, m)
		if err != nil {
			return fmt.Errorf("failed to sync project settings: %+v", err)
		}
	}

	return resourceSonarqubeProjectRead(d, m)
}

func resourceSonarqubeProjectDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/projects/delete"
	sonarQubeURL.RawQuery = url.Values{
		"project": []string{d.Get("project").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeProjectDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeProjectImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	// As per the docs, use the id to make the read work as intended (https://developer.hashicorp.com/terraform/plugin/sdkv2/resources/import)
	d.Set("project", d.Id())
	return []*schema.ResourceData{d}, nil
}
