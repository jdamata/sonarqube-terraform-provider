package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// GetQualityProfileUsergroupAssociation for unmarshalling response body from getting quality profile association
type GetQualityProfileUsergroupAssociation struct {
	Paging Paging                                          `json:"paging"`
	Groups []GetQualityProfileUsergroupAssociationProjects `json:"groups,omitempty"`
	Users  []GetQualityProfileUsergroupAssociationProjects `json:"users,omitempty"`
}

// GetQualityProfileUsergroupAssociationProjects used in GetQualityProfileUsergroupAssociation
type GetQualityProfileUsergroupAssociationProjects struct {
	Login       string `json:"login,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Selected    bool   `json:"selected"`
}

// Returns the resource represented by this file.
func resourceSonarqubeQualityProfileUsergroupAssociation() *schema.Resource {
	return &schema.Resource{
		Description: `Provides a Sonarqube Quality Profile Usergroup association resource. This can be used to associate a Quality Profile to an User or to a Group.
The feature is available on SonarQube 6.6 or newer.`,
		Create: resourceSonarqubeQualityProfileUsergroupAssociationCreate,
		Read:   resourceSonarqubeQualityProfileUsergroupAssociationRead,
		Delete: resourceSonarqubeQualityProfileUsergroupAssociationDelete,

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"login_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"login_name", "group_name"},
				Description:  "The name of the User to associate. Either `group_name` or `login_name` should be provided.",
			},
			"group_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"login_name", "group_name"},
				Description:  "The name of the Group to associate. Either `group_name` or `login_name` should be provided.",
			},
			"profile_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the Quality Profile.",
			},
			"language": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The language of the Quality Profile.",
			},
		},
	}
}

func resourceSonarqubeQualityProfileUsergroupAssociationCreate(d *schema.ResourceData, m interface{}) error {
	if err := checkProfilePermissionFeatureSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	rawQuery := url.Values{
		"qualityProfile": []string{d.Get("profile_name").(string)},
		"language":       []string{d.Get("language").(string)},
	}

	if _, ok := d.GetOk("login_name"); ok {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/add_user"
		rawQuery.Add("login", d.Get("login_name").(string))
	} else {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/add_group"
		rawQuery.Add("group", d.Get("group_name").(string))
	}

	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeQualityProfileUsergroupAssociationCreate",
	)
	defer resp.Body.Close()

	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileUsergroupAssociationCreate: Failed creating Sonarqube quality profile usergroup association for quality profile '%s': %+v", d.Get("profile_name").(string), err)
	}

	if _, ok := d.GetOk("login_name"); ok {
		d.SetId(createProfilePermissionId(d.Get("profile_name").(string), "user", d.Get("login_name").(string)))
	} else {
		d.SetId(createProfilePermissionId(d.Get("profile_name").(string), "group", d.Get("group_name").(string)))
	}
	return resourceSonarqubeQualityProfileUsergroupAssociationRead(d, m)
}

func resourceSonarqubeQualityProfileUsergroupAssociationRead(d *schema.ResourceData, m interface{}) error {
	if err := checkProfilePermissionFeatureSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.RawQuery = url.Values{
		"qualityProfile": []string{d.Get("profile_name").(string)},
		"language":       []string{d.Get("language").(string)},
		"selected":       []string{"selected"},
	}.Encode()

	if _, ok := d.GetOk("login_name"); ok {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/search_users"
	} else {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/search_groups"
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubeQualityProfileUsergroupAssociationRead",
	)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileUsergroupAssociationRead: Failed to call quality profile usergroup association api: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	QualityProfileUsergroupAssociationReadResponse := GetQualityProfileUsergroupAssociation{}
	err = json.NewDecoder(resp.Body).Decode(&QualityProfileUsergroupAssociationReadResponse)
	if err != nil {
		return fmt.Errorf("resourceQualityProfileRead: Failed to decode json into struct: %+v", err)
	}

	if _, ok := d.GetOk("login_name"); ok {
		// Loop over all groups to see if the group we need exists.
		login := d.Get("login_name").(string)
		for _, value := range QualityProfileUsergroupAssociationReadResponse.Users {
			if strings.EqualFold(value.Login, login) {
				d.Set("login_name", value.Login)
				return nil
			}
		}
	} else {
		// Loop over all groups to see if the group we need exists.
		groupName := d.Get("group_name").(string)
		for _, value := range QualityProfileUsergroupAssociationReadResponse.Groups {
			if strings.EqualFold(value.Name, groupName) {
				return nil
			}
		}
	}
	return fmt.Errorf("resourceSonarqubeQualityProfileUsergroupAssociationRead: Failed to call quality profile usergroup association api: %+v", err)
}

func resourceSonarqubeQualityProfileUsergroupAssociationDelete(d *schema.ResourceData, m interface{}) error {
	if err := checkProfilePermissionFeatureSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	rawQuery := url.Values{
		"qualityProfile": []string{d.Get("profile_name").(string)},
		"language":       []string{d.Get("language").(string)},
	}

	if _, ok := d.GetOk("login_name"); ok {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/remove_user"
		rawQuery.Add("login", d.Get("login_name").(string))
	} else {
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/remove_group"
		rawQuery.Add("group", d.Get("group_name").(string))
	}

	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeQualityProfileProjectAssociationDelete",
	)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileUsergroupAssociationDelete: Failed to call quality profile usergroup association api: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func createProfilePermissionId(profileName string, targetType string, target string) string {
	return profileName + "[" + targetType + "/" + target + "]"
}

func checkProfilePermissionFeatureSupport(conf *ProviderConfiguration) error {
	minimumVersion, _ := version.NewVersion("6.6")
	if conf.sonarQubeVersion.LessThan(minimumVersion) {
		return fmt.Errorf("Minimum required SonarQube version for quality profile permissions is %s", minimumVersion)
	}
	return nil
}
