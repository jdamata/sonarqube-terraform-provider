package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSonarqubeGroups() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get Sonarqube group resources",
		Read:        dataSourceSonarqubeGroupsRead,
		Schema: map[string]*schema.Schema{
			"search": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Search groups by name.",
			},
			"groups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the group.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The group description.",
						},
					},
				},
				Description: "The list of groups.",
			},
		},
	}
}

func dataSourceSonarqubeGroupsRead(d *schema.ResourceData, m interface{}) error {
	d.SetId(fmt.Sprintf("%d", schema.HashString(d.Get("search"))))

	groupsReadResponse, err := readGroupsFromApi(d, m)
	if err != nil {
		return err
	}

	d.Set("groups", flattenReadGroupsResponse(groupsReadResponse.Groups))

	return nil
}

func readGroupsFromApi(d *schema.ResourceData, m interface{}) (*GetGroup, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_groups/search"

	RawQuery := url.Values{
		"ps": []string{"500"},
	}

	if search, ok := d.GetOk("search"); ok {
		RawQuery.Add("q", search.(string))
	}

	sonarQubeURL.RawQuery = RawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"readGroupsFromApi",
	)
	if err != nil {
		return nil, fmt.Errorf("error reading Sonarqube groups: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	groupsReadResponse := GetGroup{}
	err = json.NewDecoder(resp.Body).Decode(&groupsReadResponse)
	if err != nil {
		return nil, fmt.Errorf("readGroupsFromApi: Failed to decode json into struct: %+v", err)
	}

	return &groupsReadResponse, nil
}

func flattenReadGroupsResponse(groups []Group) []interface{} {
	groupsList := []interface{}{}

	for _, group := range groups {
		values := map[string]interface{}{
			"name":        group.Name,
			"description": group.Description,
		}

		groupsList = append(groupsList, values)
	}

	return groupsList
}
