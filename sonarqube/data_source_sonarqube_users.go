package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSonarqubeUsers() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get Sonarqube user resources",
		Read:        dataSourceSonarqubeUsersRead,
		Schema: map[string]*schema.Schema{
			"search": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Search users by login, name and email.",
			},
			"users": {
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
						"email": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The email of the user.",
						},
						"is_local": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the user is local.",
						},
					},
				},
				Description: "The list of users.",
			},
		},
	}
}

func dataSourceSonarqubeUsersRead(d *schema.ResourceData, m interface{}) error {
	d.SetId(fmt.Sprintf("%d", schema.HashString(d.Get("search"))))

	usersReadResponse, err := readUsersFromApi(d, m)
	if err != nil {
		return err
	}

	d.Set("users", flattenReadUsersResponse(usersReadResponse.Users))

	return nil
}

func readUsersFromApi(d *schema.ResourceData, m interface{}) (*GetUser, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/users/search"

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
		"readUsersFromApi",
	)
	if err != nil {
		return nil, fmt.Errorf("error reading Sonarqube users: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	usersReadResponse := GetUser{}
	err = json.NewDecoder(resp.Body).Decode(&usersReadResponse)
	if err != nil {
		return nil, fmt.Errorf("readUsersFromApi: Failed to decode json into struct: %+v", err)
	}

	return &usersReadResponse, nil
}

func flattenReadUsersResponse(users []User) []interface{} {
	usersList := []interface{}{}

	for _, user := range users {
		values := map[string]interface{}{
			"login_name": user.Login,
			"name":       user.Name,
			"email":      user.Email,
			"is_local":   user.IsLocal,
		}

		usersList = append(usersList, values)
	}

	return usersList
}
