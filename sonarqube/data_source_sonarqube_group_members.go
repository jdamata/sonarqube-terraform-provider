package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSonarqubeGroupMembers() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get Sonarqube group member resources",
		Read:        dataSourceSonarqubeGroupMembersRead,
		Schema: map[string]*schema.Schema{
			"group": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the group.",
			},
			"login_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "To limit the search to a specific user.",
			},
			"ignore_missing": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: "If set to true, the data source will not fail if the group does not exist.",
			},
			"members": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"login_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The login name of the user.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the user.",
						},
					},
				},
				Description: "The list of members in the group.",
			},
		},
	}
}

func dataSourceSonarqubeGroupMembersRead(d *schema.ResourceData, m interface{}) error {
	search := fmt.Sprintf("%s/%s", d.Get("group").(string), d.Get("login_name").(string))
	d.SetId(fmt.Sprintf("%d", schema.HashString(search)))

	groupMembersReadResponse, err := readGroupMembersFromApi(d, m)
	if err != nil {
		return err
	}

	if groupMembersReadResponse != nil {
		d.Set("members", flattenReadGroupMembersResponse(groupMembersReadResponse.Members))
	} else {
		d.Set("members", []interface{}{})
	}

	return nil
}

func readGroupMembersFromApi(d *schema.ResourceData, m interface{}) (*GetGroupMembersResponse, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_groups/users"

	RawQuery := url.Values{
		"name": []string{d.Get("group").(string)},
		"ps":   []string{"500"},
	}

	if search, ok := d.GetOk("login_name"); ok {
		RawQuery.Add("q", search.(string))
	}

	sonarQubeURL.RawQuery = RawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"readGroupMembersFromApi",
	)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound && d.Get("ignore_missing").(bool) {
			// If the group does not exist, we don't want to fail the data source
			return nil, nil
		}
		return nil, fmt.Errorf("error reading Sonarqube group members: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	groupMembersReadResponse := GetGroupMembersResponse{}
	err = json.NewDecoder(resp.Body).Decode(&groupMembersReadResponse)
	if err != nil {
		return nil, fmt.Errorf("readGroupMembersFromApi: Failed to decode json into struct: %+v", err)
	}

	return &groupMembersReadResponse, nil
}

func flattenReadGroupMembersResponse(members []GroupMember) []interface{} {
	membersList := []interface{}{}

	for _, member := range members {
		values := map[string]interface{}{
			"login_name": member.LoginName,
			"name":       member.Name,
		}

		membersList = append(membersList, values)
	}

	return membersList
}
