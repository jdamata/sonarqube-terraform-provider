package sonarqube

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// CreatePermissionTemplateResponse struct
type CreatePermissionTemplateResponse struct {
	PermissionTemplate PermissionTemplate `json:"permissionTemplate"`
}

// GetPermissionTemplates struct
type GetPermissionTemplates struct {
	Paging              Paging               `json:"paging"`
	PermissionTemplates []PermissionTemplate `json:"permissionTemplates"`
}

// PermissionTemplate struct
type PermissionTemplate struct {
	ID                string `json:"id,omitempty"`
	Name              string `json:"name,omitempty"`
	Description       string `json:"description,omitempty"`
	ProjectKeyPattern string `json:"projectKeyPattern,omitempty"`
}

// Returns the resource represented by this file.
func resourceSonarqubePermissionTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubePermissionTemplateCreate,
		Read:   resourceSonarqubePermissionTemplateRead,
		Update: resourceSonarqubePermissionTemplateUpdate,
		Delete: resourceSonarqubePermissionTemplateDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubePermissionTemplateImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"project_key_pattern": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"default": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceSonarqubePermissionTemplateCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/create_template"
	sonarQubeURL.RawQuery = url.Values{
		"name":              []string{d.Get("name").(string)},
		"description":       []string{d.Get("description").(string)},
		"projectKeyPattern": []string{d.Get("project_key_pattern").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubePermissionTemplateCreate",
	)
	if err != nil {
		return fmt.Errorf("error creating Sonarqube permission template: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	permissionTemplateResponse := CreatePermissionTemplateResponse{}
	err = json.NewDecoder(resp.Body).Decode(&permissionTemplateResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubePermissionTemplateCreate: Failed to decode json into struct: %+v", err)
	}

	if permissionTemplateResponse.PermissionTemplate.ID != "" {
		d.SetId(permissionTemplateResponse.PermissionTemplate.ID)
	} else {
		return fmt.Errorf("resourceSonarqubePermissionTemplateCreate: Create response didn't contain an ID")
	}

	// If default is set to true, set this permission template as the default.
	if d.Get("default").(bool) {
		err = resourceSonarqubePermissionTemplateSetDefault(sonarQubeURL, d.Id(), m)
		if err != nil {
			return err
		}
	}

	return resourceSonarqubePermissionTemplateRead(d, m)
}

func resourceSonarqubePermissionTemplateRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/search_templates"
	sonarQubeURL.RawQuery = url.Values{
		"q": []string{d.Get("name").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubePermissionTemplateRead",
	)
	if err != nil {
		return fmt.Errorf("error reading Sonarqube permission templates: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	permissionTemplateReadResponse := GetPermissionTemplates{}
	err = json.NewDecoder(resp.Body).Decode(&permissionTemplateReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubePermissionTemplateRead: Failed to decode json into struct: %+v", err)
	}

	// Loop over all permission templates to see if the template we look for exists.
	for _, value := range permissionTemplateReadResponse.PermissionTemplates {
		log.Printf("[DEBUG][resourceSonarqubePermissionTemplateRead] Comparing '%s' with '%s'", d.Id(), value.ID)
		if d.Id() == value.ID {
			log.Printf("[DEBUG][resourceSonarqubePermissionTemplateRead] Found PermissionTemplate with ID '%s'", value.ID)
			// If it does, set the values of that template
			d.SetId(value.ID)
			d.Set("name", value.Name)
			d.Set("description", value.Description)
			d.Set("project_key_pattern", value.ProjectKeyPattern)
			return nil
		}
	}

	return fmt.Errorf("resourceSonarqubePermissionTemplateRead: Failed to find template with ID: %+v", d.Id())

}

func resourceSonarqubePermissionTemplateUpdate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/update_template"

	rawQuery := url.Values{
		"id": []string{d.Id()},
	}

	if _, ok := d.GetOk("description"); ok {
		rawQuery.Add("description", d.Get("description").(string))
	} else {
		rawQuery.Add("description", "")
	}

	if _, ok := d.GetOk("project_key_pattern"); ok {
		rawQuery.Add("projectKeyPattern", d.Get("project_key_pattern").(string))
	} else {
		rawQuery.Add("projectKeyPattern", "")
	}

	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubePermissionTemplateUpdate",
	)
	if err != nil {
		return fmt.Errorf("error updating Sonarqube permission template: %+v", err)
	}
	defer resp.Body.Close()

	// If default is set to true, set this permission template as the default.
	if d.Get("default").(bool) {
		err = resourceSonarqubePermissionTemplateSetDefault(sonarQubeURL, d.Id(), m)
		if err != nil {
			return err
		}
	}

	return resourceSonarqubePermissionTemplateRead(d, m)
}

func resourceSonarqubePermissionTemplateDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/delete_template"
	sonarQubeURL.RawQuery = url.Values{
		"templateId": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubePermissionTemplateDelete",
	)
	if err != nil {
		return fmt.Errorf("error deleting Sonarqube permission template: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubePermissionTemplateImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubePermissionTemplateRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func resourceSonarqubePermissionTemplateSetDefault(sonarQubeURL url.URL, templateID string, m interface{}) error {
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/set_default_template"
	sonarQubeURL.RawQuery = url.Values{
		"templateId": []string{templateID},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubePermissionTemplateCreate",
	)
	if err != nil {
		return fmt.Errorf("error setting Sonarqube permission template to default: %+v", err)
	}
	defer resp.Body.Close()
	return nil
}
