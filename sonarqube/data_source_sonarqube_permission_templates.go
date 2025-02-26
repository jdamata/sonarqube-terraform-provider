package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSonarqubePermissionTemplates() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get Sonarqube permission templates resources",
		Read:        dataSourceSonarqubePermissionTemplatesRead,
		Schema: map[string]*schema.Schema{
			"search": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Search permission templates by name.",
			},
			"permission_templates": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The id of the permission template.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the permission template.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the permission template.",
						},
						"project_key_pattern": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The project key pattern of the permission template.",
						},
					},
				},
				Description: "The list of permission templates.",
			},
		},
	}
}

func dataSourceSonarqubePermissionTemplatesRead(d *schema.ResourceData, m interface{}) error {
	d.SetId(fmt.Sprintf("%d", schema.HashString(d.Get("search"))))

	permissionTemplatesReadResponse, err := readPermissionTemplatesFromApi(d, m)
	if err != nil {
		return err
	}

	d.Set("permission_templates", flattenReadPermissionTemplatesResponse(permissionTemplatesReadResponse.PermissionTemplates))

	return nil
}

func readPermissionTemplatesFromApi(d *schema.ResourceData, m interface{}) (*GetPermissionTemplates, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/search_templates"

	RawQuery := url.Values{}
	if search, ok := d.GetOk("search"); ok {
		RawQuery.Add("q", search.(string))
	}

	sonarQubeURL.RawQuery = RawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"readPermissionTemplatesFromApi",
	)
	if err != nil {
		return nil, fmt.Errorf("error reading Sonarqube permission templates: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	permissionTemplatesReadResponse := GetPermissionTemplates{}
	err = json.NewDecoder(resp.Body).Decode(&permissionTemplatesReadResponse)
	if err != nil {
		return nil, fmt.Errorf("readPermissionTemplatesFromApi: Failed to decode json into struct: %+v", err)
	}

	return &permissionTemplatesReadResponse, nil
}

func flattenReadPermissionTemplatesResponse(permissionTemplates []PermissionTemplate) []interface{} {
	permissionTemplatesList := []interface{}{}

	for _, permissionTemplate := range permissionTemplates {
		values := map[string]interface{}{
			"id":                  permissionTemplate.ID,
			"name":                permissionTemplate.Name,
			"description":         permissionTemplate.Description,
			"project_key_pattern": permissionTemplate.ProjectKeyPattern,
		}

		permissionTemplatesList = append(permissionTemplatesList, values)
	}

	return permissionTemplatesList
}
