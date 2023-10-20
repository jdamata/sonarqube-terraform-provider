package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// GroupMember struct
type GroupMember struct {
	LoginName string `json:"login,omitempty"`
	Name      string `json:"name,omitempty"`
}

// GetGroupMembersResponse for unmarshalling response body of group creation
type GetGroupMembersResponse struct {
	Paging  Paging        `json:"paging"`
	Members []GroupMember `json:"users"`
}

// Returns the resource represented by this file.
func resourceSonarqubeGroupMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeGroupMemberCreate,
		Read:   resourceSonarqubeGroupMemberRead,
		Delete: resourceSonarqubeGroupMemberDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeGroupMemberImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"login_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSonarqubeGroupMemberCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_groups/add_user"
	sonarQubeURL.RawQuery = url.Values{
		"name":  []string{d.Get("name").(string)},
		"login": []string{d.Get("login_name").(string)},
	}.Encode()

	groupMembershipId := createGroupMembershipId(d.Get("name").(string), d.Get("login_name").(string))

	// We need to check if a user is already a member in advance because SQ does not report this conflict in the add_user API call:
	exists, _ := checkGroupMemberExists(d.Get("name").(string), d.Get("login_name").(string), m)
	if exists {
		return fmt.Errorf("resourceSonarqubeGroupMemberCreate: Group membership already exists: %+v", groupMembershipId)
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeGroupMemberCreate",
	)
	if err != nil {
		return fmt.Errorf("error adding user '%s' to Sonarqube group '%s': %w", d.Get("login_name").(string), d.Get("name").(string), err)
	}
	defer resp.Body.Close()

	d.SetId(groupMembershipId)

	return resourceSonarqubeGroupMemberRead(d, m)
}

func resourceSonarqubeGroupMemberRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_groups/users"
	sonarQubeURL.RawQuery = url.Values{
		"name": []string{d.Get("name").(string)},
		"q":    []string{d.Get("login_name").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeGroupMemberRead",
	)
	if err != nil {
		return fmt.Errorf("error reading Sonarqube members of group '%s': %w", d.Get("name").(string), err)
	}
	defer resp.Body.Close()

	readSuccess := false
	// Decode response into struct
	groupMemberReadResponse := GetGroupMembersResponse{}
	err = json.NewDecoder(resp.Body).Decode(&groupMemberReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeGroupRead: Failed to decode json into struct: %+v", err)
	}
	// Loop over all returned members to see if the member we need exists.
	for _, value := range groupMemberReadResponse.Members {
		if d.Get("login_name").(string) == value.LoginName {
			// If it does, set the values of that group membership
			d.SetId(createGroupMembershipId(d.Get("name").(string), d.Get("login_name").(string)))
			d.Set("name", d.Get("name").(string))
			d.Set("login_name", value.LoginName)
			readSuccess = true
			break
		}
	}

	if !readSuccess {
		// Group member not found
		d.SetId("")
	}

	return nil
}

func resourceSonarqubeGroupMemberDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_groups/remove_user"

	sonarQubeURL.RawQuery = url.Values{
		"name":  []string{d.Get("name").(string)},
		"login": []string{d.Get("login_name").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeGroupMemberDelete",
	)
	if err != nil {
		return fmt.Errorf("error deleting Sonarqube member '%s' from group '%s': %w", d.Get("login_name").(string), d.Get("name").(string), err)
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeGroupMemberImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	rgx := regexp.MustCompile(`(.*?)\[(.*?)\]`)
	rs := rgx.FindStringSubmatch(d.Id())
	groupName := rs[1]
	loginName := rs[2]

	exists, _ := checkGroupMemberExists(groupName, loginName, m)
	if exists {
		d.Set("name", groupName)
		d.Set("login_name", loginName)

		return []*schema.ResourceData{d}, nil
	} else {
		return nil, fmt.Errorf("User '%s' not a member of group '%s'", loginName, groupName)
	}
}

func checkGroupMemberExists(groupName string, loginName string, m interface{}) (bool, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_groups/users"
	sonarQubeURL.RawQuery = url.Values{
		"name": []string{groupName},
		"q":    []string{loginName},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"checkGroupMemberExists",
	)
	if err != nil {
		return false, fmt.Errorf("error reading Sonarqube members of group '%s': %w", groupName, err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	groupMemberReadResponse := GetGroupMembersResponse{}
	err = json.NewDecoder(resp.Body).Decode(&groupMemberReadResponse)
	if err != nil {
		return false, fmt.Errorf("checkGroupMemberExists: Failed to decode json into struct: %w", err)
	}
	// Loop over all returned members to see if the member we need exists.
	for _, value := range groupMemberReadResponse.Members {
		if loginName == value.LoginName {
			return true, nil
		}
	}

	return false, nil
}

func createGroupMembershipId(groupName string, loginName string) string {
	return groupName + "[" + loginName + "]"
}
