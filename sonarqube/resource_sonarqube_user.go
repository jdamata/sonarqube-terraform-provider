package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// User struct
type User struct {
	Login       string   `json:"login,omitempty"`
	Name        string   `json:"name,omitempty"`
	Email       string   `json:"email,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	IsActive    bool     `json:"active,omitempty"`
	IsLocal     bool     `json:"local,omitempty"`
}

// GetUser for unmarshalling response body where users are retured
type GetUser struct {
	Paging Paging `json:"paging"`
	Users  []User `json:"users"`
}

// CreateUserResponse struct
type CreateUserResponse struct {
	User User `json:"user"`
}

// Returns the resource represented by this file.
func resourceSonarqubeUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeUserCreate,
		Read:   resourceSonarqubeUserRead,
		Update: resourceSonarqubeUserUpdate,
		Delete: resourceSonarqubeUserDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeUserImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"login_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"email": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"is_local": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true,
			},
		},
	}
}

func resourceSonarqubeUserCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/users/create"

	isLocal := d.Get("is_local").(bool)

	rawQuery := url.Values{
		"login": []string{d.Get("login_name").(string)},
		"name":  []string{d.Get("name").(string)},
		"local": []string{strconv.FormatBool(isLocal)},
	}

	if password, ok := d.GetOk("password"); ok {
		rawQuery.Add("password", password.(string))
	}

	if email, ok := d.GetOk("email"); ok {
		rawQuery.Add("email", email.(string))
	}

	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeUserCreate",
	)
	if err != nil {
		return fmt.Errorf("error creating Sonarqube user: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	userResponse := CreateUserResponse{}
	err = json.NewDecoder(resp.Body).Decode(&userResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeUserCreate: Failed to decode json into struct: %+v", err)
	}

	if userResponse.User.Login != "" {
		d.SetId(userResponse.User.Login)
	} else {
		return fmt.Errorf("resourceSonarqubeUserCreate: Create response didn't contain the user login")
	}

	return resourceSonarqubeUserRead(d, m)
}

func resourceSonarqubeUserRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/users/search"

	sonarQubeURL.RawQuery = url.Values{
		"ps": []string{"500"},
		"q":  []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeUserRead",
	)
	if err != nil {
		return fmt.Errorf("error reading Sonarqube user: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	userResponse := GetUser{}
	err = json.NewDecoder(resp.Body).Decode(&userResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeUserCreate: Failed to decode json into struct: %+v", err)
	}

	// Loop over all users to see if the current user exists.
	for _, value := range userResponse.Users {
		if d.Id() == value.Login {
			d.SetId(value.Login)
			d.Set("login_name", value.Login)
			d.Set("name", value.Name)
			d.Set("email", value.Email)
			d.Set("is_local", value.IsLocal)
			return nil
		}
	}

	return fmt.Errorf("resourceSonarqubeUserRead: Failed to find user: %+v", d.Id())
}

func resourceSonarqubeUserUpdate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURLSubPath := sonarQubeURL.Path
	// handle default updates (api/users/update)
	if d.HasChange("email") {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURLSubPath, "/") + "/api/users/update"
		sonarQubeURL.RawQuery = url.Values{
			"login": []string{d.Id()},
			"email": []string{d.Get("email").(string)},
		}.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL,
			http.StatusOK,
			"resourceSonarqubeUserUpdate",
		)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube user: %+v", err)
		}
		defer resp.Body.Close()
	}

	// handle password updates (api/users/change_password)
	if d.HasChange("password") {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURLSubPath, "/") + "/api/users/change_password"
		sonarQubeURL.RawQuery = url.Values{
			"login":    []string{d.Id()},
			"password": []string{d.Get("password").(string)},
		}.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL,
			http.StatusNoContent,
			"resourceSonarqubeUserUpdate",
		)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube user: %+v", err)
		}
		defer resp.Body.Close()
	}

	return resourceSonarqubeUserRead(d, m)
}

func resourceSonarqubeUserDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/users/deactivate"
	sonarQubeURL.RawQuery = url.Values{
		"login":     []string{d.Id()},
		"anonymize": []string{strconv.FormatBool(m.(*ProviderConfiguration).sonarQubeAnonymizeUsers)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeUserDelete",
	)
	if err != nil {
		return fmt.Errorf("error deleting (deactivating) Sonarqube user: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeUserImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubeUserRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
