package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// GetAlmAzure for unmarshalling response body from alm list definitions. With only azure populated
type GetAlmAzure struct {
	Azure []struct {
		Key string `json:"key"`
		URL string `json:"url"`
	} `json:"azure"`
}

// Returns the resource represented by this file.
func resourceSonarqubeAlmAzure() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeAlmAzureCreate,
		Read:   resourceSonarqubeAlmAzureRead,
		Update: resourceSonarqubeAlmAzureUpdate,
		Delete: resourceSonarqubeAlmAzureDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeAlmAzureImport,
		},
		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"key": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Unique key of the Azure Devops instance setting",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 200)),
			},
			"personal_access_token": {
				Type:             schema.TypeString,
				Required:         true,
				Sensitive:        true,
				Description:      "Azure Devops personal access token",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 2000)),
			},
			"url": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Azure API URL",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 2000)),
			},
		},
	}
}

func resourceSonarqubeAlmAzureCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/create_azure"

	sonarQubeURL.RawQuery = url.Values{
		"key":                 []string{d.Get("key").(string)},
		"personalAccessToken": []string{d.Get("personal_access_token").(string)},
		"url":                 []string{d.Get("url").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeAlmAzureCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d.SetId(d.Get("key").(string))

	return resourceSonarqubeAlmAzureRead(d, m)
}

func resourceSonarqubeAlmAzureRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/list_definitions"

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeAlmAzureRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	AlmAzureReadResponse := GetAlmAzure{}
	err = json.NewDecoder(resp.Body).Decode(&AlmAzureReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeAlmAzureRead: Failed to decode json into struct: %+v", err)
	}
	// Loop over all Azure instances to see if the Alm instance exists.
	for _, value := range AlmAzureReadResponse.Azure {
		if d.Id() == value.Key {
			d.Set("key", value.Key)
			d.Set("url", value.URL)
			return nil
		}
	}
	return fmt.Errorf("resourceSonarqubeAzureBindingRead: Failed to find azure binding: %+v", d.Id())

}
func resourceSonarqubeAlmAzureUpdate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/update_azure"
	sonarQubeURL.RawQuery = url.Values{
		"key":                 []string{d.Id()},
		"newKey":              []string{d.Get("key").(string)},
		"personalAccessToken": []string{d.Get("personal_access_token").(string)},
		"url":                 []string{d.Get("url").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeAlmAzureUpdate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return resourceSonarqubeAlmAzureRead(d, m)
}

func resourceSonarqubeAlmAzureDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/delete"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{d.Get("key").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeAlmAzureDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeAlmAzureImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	// import id in format {key}/{personal_access_token}
	importIdComponents := strings.SplitN(d.Id(), "/", 2)

	if len(importIdComponents) != 2 {
		return nil, fmt.Errorf("resourceSonarqubeAlmAzureImport: Import id: '%+v' is not in format {key}/{personal_access_token}", d.Id())
	}

	// set Id to key for Read
	d.SetId(importIdComponents[0])
	if err := resourceSonarqubeAlmAzureRead(d, m); err != nil {
		return nil, err
	}

	// Add personal_access_token from import id
	d.Set("personal_access_token", importIdComponents[1])

	return []*schema.ResourceData{d}, nil
}
