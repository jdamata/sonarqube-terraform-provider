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

// Returns the resource represented by this file.
func resourceSonarqubeGitlabBinding() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeGitlabBindingCreate,
		// You can update any project binding with the same API call as the CREATE
		Update: resourceSonarqubeGitlabBindingCreate,
		Read:   resourceSonarqubeGitlabBindingRead,
		Delete: resourceSonarqubeGitlabBindingDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeGitlabBindingImport,
		},
		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"alm_setting": {
				Type:     schema.TypeString,
				Required: true,
			},
			"monorepo": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "false",
			},
			"project": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func checkGitlabBindingSupport(conf *ProviderConfiguration) error {
	if strings.ToLower(conf.sonarQubeEdition) == "community" {
		return fmt.Errorf("GitLab Bindings are not supported in the Community edition of SonarQube. You are using: SonarQube %s version %s", conf.sonarQubeEdition, conf.sonarQubeVersion)
	}
	return nil
}

func resourceSonarqubeGitlabBindingCreate(d *schema.ResourceData, m interface{}) error {
	if err := checkGitlabBindingSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/set_gitlab_binding"

	sonarQubeURL.RawQuery = url.Values{
		"almSetting": []string{d.Get("alm_setting").(string)},
		"monorepo":   []string{d.Get("monorepo").(string)},
		"project":    []string{d.Get("project").(string)},
		"repository": []string{d.Get("repository").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeGitlabBindingCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	id := fmt.Sprintf("%v/%v", d.Get("project").(string), d.Get("repository").(string))
	d.SetId(id)

	return resourceSonarqubeGitlabBindingRead(d, m)
}

func resourceSonarqubeGitlabBindingRead(d *schema.ResourceData, m interface{}) error {
	if err := checkGitlabBindingSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	idSlice := strings.SplitN(d.Id(), "/", 2)
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/get_binding"
	sonarQubeURL.RawQuery = url.Values{
		"project": []string{idSlice[0]},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeGitlabBindingRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	BindingReadResponse := GetBinding{}
	err = json.NewDecoder(resp.Body).Decode(&BindingReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeGitlabBindingRead: Failed to decode json into struct: %+v", err)
	}
	// Loop over all branches to see if the main branch we need exists.
	if idSlice[1] == BindingReadResponse.Repository && BindingReadResponse.Alm == "gitlab" {
		d.Set("project", idSlice[0])
		d.Set("repository", idSlice[1])
		d.Set("alm_setting", BindingReadResponse.Key)
		d.Set("monorepo", strconv.FormatBool(BindingReadResponse.Monorepo))

		return nil
	}
	return fmt.Errorf("resourceSonarqubeGitlabBindingRead: Failed to find gitlab binding: %+v", d.Id())
}

func resourceSonarqubeGitlabBindingDelete(d *schema.ResourceData, m interface{}) error {
	if err := checkGitlabBindingSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/delete_binding"
	sonarQubeURL.RawQuery = url.Values{
		"project": []string{d.Get("project").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeGitlabBindingDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeGitlabBindingImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubeGitlabBindingRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
