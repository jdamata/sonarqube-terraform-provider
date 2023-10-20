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

// GetBinding for unmarshalling response body from getting project binding details
type GetBinding struct {
	Key                   string `json:"key"`
	Alm                   string `json:"alm"`
	Repository            string `json:"repository"`
	URL                   string `json:"url"`
	SummaryCommentEnabled bool   `json:"summaryCommentEnabled,omitempty"`
	Monorepo              bool   `json:"monorepo"`
}

// Returns the resource represented by this file.
func resourceSonarqubeGithubBinding() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeGithubBindingCreate,
		Read:   resourceSonarqubeGithubBindingRead,
		Delete: resourceSonarqubeGithubBindingDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeGithubBindingImport,
		},
		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"alm_setting": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"monorepo": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "false",
				ForceNew: true,
			},
			"project": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"summary_comment_enabled": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "true",
				ForceNew: true,
			},
		},
	}
}

func checkGithubBindingSupport(conf *ProviderConfiguration) error {
	if strings.ToLower(conf.sonarQubeEdition) == "community" {
		return fmt.Errorf("GitHub Bindings are not supported in the Community edition of SonarQube. You are using: SonaQube %s version %s", conf.sonarQubeEdition, conf.sonarQubeVersion)
	}
	return nil
}

func resourceSonarqubeGithubBindingCreate(d *schema.ResourceData, m interface{}) error {
	if err := checkGithubBindingSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/set_github_binding"

	sonarQubeURL.RawQuery = url.Values{
		"almSetting":            []string{d.Get("alm_setting").(string)},
		"monorepo":              []string{d.Get("monorepo").(string)},
		"project":               []string{d.Get("project").(string)},
		"repository":            []string{d.Get("repository").(string)},
		"summaryCommentEnabled": []string{d.Get("summary_comment_enabled").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeGithubBindingCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	id := fmt.Sprintf("%v/%v", d.Get("project").(string), d.Get("repository").(string))
	d.SetId(id)

	return resourceSonarqubeGithubBindingRead(d, m)
}

func resourceSonarqubeGithubBindingRead(d *schema.ResourceData, m interface{}) error {
	if err := checkGithubBindingSupport(m.(*ProviderConfiguration)); err != nil {
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
		"resourceSonarqubeGithubBindingRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	BindingReadResponse := GetBinding{}
	err = json.NewDecoder(resp.Body).Decode(&BindingReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeGithubBindingRead: Failed to decode json into struct: %+v", err)
	}
	// Loop over all branches to see if the main branch we need exists.
	if idSlice[1] == BindingReadResponse.Repository && BindingReadResponse.Alm == "github" {
		d.Set("project", idSlice[0])
		d.Set("repository", idSlice[1])
		d.Set("alm_setting", BindingReadResponse.Key)
		d.Set("monorepo", strconv.FormatBool(BindingReadResponse.Monorepo))
		d.Set("summary_comment_enabled", strconv.FormatBool(BindingReadResponse.SummaryCommentEnabled))

		return nil
	}
	return fmt.Errorf("resourceSonarqubeGithubBindingRead: Failed to find github binding: %+v", d.Id())
}

func resourceSonarqubeGithubBindingDelete(d *schema.ResourceData, m interface{}) error {
	if err := checkGithubBindingSupport(m.(*ProviderConfiguration)); err != nil {
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
		"resourceSonarqubeGithubBindingDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeGithubBindingImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubeGithubBindingRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
