package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// GetInstalledPlugins for unmarshalling response body from geting installed plugins
type GetInstalledPlugins struct {
	Plugins []Plugin `json:"plugins"`
}

// Plugin used in GetInstalledPlugins
type Plugin struct {
	Key                 string `json:"key"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Version             string `json:"version"`
	License             string `json:"license"`
	OrganizationName    string `json:"organizationName"`
	OrganizationURL     string `json:"organizationUrl"`
	EditionBundled      bool   `json:"editionBundled"`
	HomepageURL         string `json:"homepageUrl"`
	IssueTrackerURL     string `json:"issueTrackerUrl"`
	ImplementationBuild string `json:"implementationBuild"`
	Filename            string `json:"filename"`
	Hash                string `json:"hash"`
	SonarLintSupported  bool   `json:"sonarLintSupported"`
	DocumentationPath   string `json:"documentationPath"`
	UpdatedAt           int    `json:"updatedAt"`
}

// Returns the resource represented by this file.
func resourceSonarqubePlugin() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubePluginCreate,
		Read:   resourceSonarqubePluginRead,
		Delete: resourceSonarqubePluginDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubePluginImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSonarqubePluginCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/plugins/install"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{d.Get("key").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubePluginCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d.SetId(d.Get("key").(string))
	return resourceSonarqubePluginRead(d, m)
}

func resourceSonarqubePluginRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/plugins/installed"

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubePluginRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	getInstalledPlugins := GetInstalledPlugins{}
	err = json.NewDecoder(resp.Body).Decode(&getInstalledPlugins)
	if err != nil {
		return fmt.Errorf("resourceSonarqubePluginRead: Failed to decode json into struct: %+v", err)
	}

	// Loop over all projects to see if the project we need exists.
	for _, value := range getInstalledPlugins.Plugins {
		if d.Id() == value.Key {
			// If it does, set the values of that project
			d.SetId(value.Key)
			d.Set("key", value.Key)
			return nil
		}
	}

	return fmt.Errorf("resourceSonarqubePluginRead: Failed to find plugin: %+v", d.Id())
}

func resourceSonarqubePluginDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/plugins/uninstall"

	sonarQubeURL.RawQuery = url.Values{
		"key": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubePluginDelete",
	)
	if err != nil {
		return fmt.Errorf("resourceSonarqubePluginDelete: Failed to delete plugin: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubePluginImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubePluginRead(d, m); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
