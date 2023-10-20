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

// QualityProfile struct
type QualityProfile struct {
	IsDefault    bool   `json:"isDefault,omitempty"`
	IsInherited  bool   `json:"isInherited,omitempty"`
	Language     string `json:"language"`
	LanguageName string `json:"languageName"`
	Name         string `json:"name"`
	Key          string `json:"key"`
}

// CreateQualityProfileResponse for unmarshalling response body from creating quality profiles
type CreateQualityProfileResponse struct {
	Profile  QualityProfile `json:"profile"`
	Warnings []string       `json:"warnings"`
}

// GetQualityProfileList for unmarshalling response body of quality gate get
type GetQualityProfileList struct {
	Profiles []GetQualityProfile `json:"profiles"`
}

// GetQualityProfile for unmarshalling response body of quality gate get
type GetQualityProfile struct {
	Key                       string                   `json:"key"`
	Name                      string                   `json:"name"`
	Language                  string                   `json:"language"`
	LanguageName              string                   `json:"languageName"`
	IsInherited               bool                     `json:"isInherited"`
	IsBuiltIn                 bool                     `json:"isBuiltIn"`
	ActiveRuleCount           int                      `json:"activeRuleCount"`
	ActiveDeprecatedRuleCount int                      `json:"activeDeprecatedRuleCount"`
	IsDefault                 bool                     `json:"isDefault"`
	RuleUpdatedAt             string                   `json:"ruleUpdatedAt"`
	LastUsed                  string                   `json:"lastUsed"`
	Actions                   GetQualityProfileActions `json:"actions"`
}

// GetQualityProfileActions for unmarshalling response body of quality gate get
type GetQualityProfileActions struct {
	Edit              bool `json:"edit"`
	SetAsDefault      bool `json:"setAsDefault"`
	Copy              bool `json:"copy"`
	Delete            bool `json:"delete"`
	AssociateProjects bool `json:"associateProjects"`
}

// Returns the resource represented by this file.
func resourceSonarqubeQualityProfile() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeQualityProfileCreate,
		Read:   resourceSonarqubeQualityProfileRead,
		Delete: resourceSonarqubeQualityProfileDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeQualityProfileImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Quality profile name",
				ValidateDiagFunc: validation.ToDiagFunc(
					validation.StringLenBetween(0, 100),
				),
			},
			"key": {
				Type:        schema.TypeString,
				Description: "Quality profile key",
				Computed:    true,
			},
			"language": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Quality profile language",
			},
			"is_default": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Is the default profile",
				Default:     false,
				ForceNew:    true,
			},
			"parent": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSonarqubeQualityProfileCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/create"

	sonarQubeURL.RawQuery = url.Values{
		"name":     []string{d.Get("name").(string)},
		"language": []string{d.Get("language").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeQualityProfileCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	qualityProfileResponse := CreateQualityProfileResponse{}
	err = json.NewDecoder(resp.Body).Decode(&qualityProfileResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileCreate: Failed to decode json into struct: %+v", err)
	}

	if d.Get("is_default").(bool) {
		err := setDefaultQualityProfile(d, m, d.Get("is_default").(bool))
		if err != nil {
			return err
		}
	}
	err = setParentQualityProfile(d, m)
	if err != nil {
		return err
	}

	d.SetId(qualityProfileResponse.Profile.Key)
	return resourceSonarqubeQualityProfileRead(d, m)
}

func resourceSonarqubeQualityProfileRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/search"

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeQualityProfileRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	getQualityProfileResponse := GetQualityProfileList{}
	err = json.NewDecoder(resp.Body).Decode(&getQualityProfileResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileRead: Failed to decode json into struct: %+v", err)
	}

	for _, value := range getQualityProfileResponse.Profiles {
		if d.Id() == value.Key {
			d.SetId(value.Key)
			d.Set("name", value.Name)
			d.Set("language", value.Language)
			d.Set("key", value.Key)
			d.Set("is_default", value.IsDefault)
			return nil
		}
	}

	return fmt.Errorf("resourceSonarqubeQualityProfileRead: Failed to find project: %+v", d.Id())
}

func resourceSonarqubeQualityProfileDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/delete"

	sonarQubeURL.RawQuery = url.Values{
		"qualityProfile": []string{d.Get("name").(string)},
		"language":       []string{d.Get("language").(string)},
	}.Encode()

	err := setDefaultQualityProfile(d, m, false)

	if err != nil {
		return err
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeQualityProfileDelete",
	)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileDelete: Failed to delete quality profile: %+v", err)
	}
	defer resp.Body.Close()

	return nil

}

func resourceSonarqubeQualityProfileImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubeQualityProfileRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func setDefaultQualityProfile(d *schema.ResourceData, m interface{}, setDefault bool) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/set_default"

	if setDefault {
		sonarQubeURL.RawQuery = url.Values{
			"qualityProfile": []string{d.Get("name").(string)},
			"language":       []string{d.Get("language").(string)},
		}.Encode()
	} else {
		sonarQubeURL.RawQuery = url.Values{
			"qualityProfile": []string{"Sonar way"},
			"language":       []string{d.Get("language").(string)},
		}.Encode()
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"setDefaultQualityProfile",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func setParentQualityProfile(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/change_parent"

	sonarQubeURL.RawQuery = url.Values{
		"qualityProfile":       []string{d.Get("name").(string)},
		"language":             []string{d.Get("language").(string)},
		"parentQualityProfile": []string{d.Get("parent").(string)},
	}.Encode()
	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"setParentQualityProfile",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
