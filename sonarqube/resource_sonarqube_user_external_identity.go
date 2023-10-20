package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Returns the resource represented by this file.
func resourceSonarqubeUserExternalIdentity() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeUserExternalIdentityCreate,
		Read:   resourceSonarqubeUserExternalIdentityRead,
		Delete: resourceSonarqubeUserExternalIdentityDelete,

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"login_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"external_identity": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"external_provider": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSonarqubeUserExternalIdentityCreate(d *schema.ResourceData, m interface{}) error {
	isLocal, err := isLocal(d.Get("login_name").(string), m)
	if err != nil {
		return fmt.Errorf("Error updating Sonarqube user: %+v", err)
	}
	if isLocal {
		return fmt.Errorf("Error setting external identity: Sonarqube user '%+v' is not 'external'", d.Get("login_name").(string))
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/users/update_identity_provider"

	rawQuery := url.Values{
		"login":               []string{d.Get("login_name").(string)},
		"newExternalIdentity": []string{d.Get("external_identity").(string)},
		"newExternalProvider": []string{d.Get("external_provider").(string)},
	}

	sonarQubeURL.RawQuery = rawQuery.Encode()

	_, err = httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeUserExternalIdentityCreate",
	)
	if err != nil {
		return fmt.Errorf("Error updating Sonarqube user: %+v", err)
	}

	d.SetId(d.Get("login_name").(string))
	d.Set("external_identity", d.Get("external_identity").(string))
	d.Set("external_provider", d.Get("external_provider").(string))

	return nil
}

func resourceSonarqubeUserExternalIdentityRead(d *schema.ResourceData, m interface{}) error {

	return nil
}

func resourceSonarqubeUserExternalIdentityDelete(d *schema.ResourceData, m interface{}) error {

	return nil
}

func isLocal(login string, m interface{}) (bool, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/users/search"

	sonarQubeURL.RawQuery = url.Values{
		"q": []string{login},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeUserExternalIdentity",
	)
	if err != nil {
		return false, fmt.Errorf("Error reading Sonarqube user: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	userResponse := GetUser{}
	err = json.NewDecoder(resp.Body).Decode(&userResponse)
	if err != nil {
		return false, fmt.Errorf("Failed to decode json into struct: %+v", err)
	}

	// Loop over all users to find the requested user
	for _, value := range userResponse.Users {
		if login == value.Login {
			return value.IsLocal, nil
		}
	}

	// User not found in response
	return false, fmt.Errorf("Failed to find user: %+v", login)
}
