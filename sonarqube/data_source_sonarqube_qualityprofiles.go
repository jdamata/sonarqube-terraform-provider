package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSonarqubeQualityProfiles() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get Sonarqube quality profiles resources",
		Read:        dataSourceSonarqubeQualityProfilesRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Search quality profiles by name.",
			},
			"language": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Search quality profiles by language.",
			},
			"quality_profiles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The key of the Quality Profile.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the Quality Profile.",
						},
						"language": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The language of the Quality Profile.",
						},
						"is_default": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the Quality Profile is default.",
						},
					},
				},
				Description: "The list of quality profiles.",
			},
		},
	}
}

func dataSourceSonarqubeQualityProfilesRead(d *schema.ResourceData, m interface{}) error {
	search := fmt.Sprintf("%s/%s", d.Get("name").(string), d.Get("language").(string))
	d.SetId(fmt.Sprintf("%d", schema.HashString(search)))

	qualityProfilesReadResponse, err := readQualityProfilesFromApi(d, m)
	if err != nil {
		return err
	}

	d.Set("quality_profiles", flattenReadQualityProfilesResponse(qualityProfilesReadResponse.Profiles))

	return nil
}

func readQualityProfilesFromApi(d *schema.ResourceData, m interface{}) (*GetQualityProfileList, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/search"

	RawQuery := url.Values{}
	if name, ok := d.GetOk("name"); ok {
		RawQuery.Add("qualityProfile", name.(string))
	}
	if language, ok := d.GetOk("language"); ok {
		RawQuery.Add("language", language.(string))
	}

	sonarQubeURL.RawQuery = RawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"readQualityProfilesFromApi",
	)
	if err != nil {
		return nil, fmt.Errorf("error reading Sonarqube quality profiles: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	qualityProfilesReadResponse := GetQualityProfileList{}
	err = json.NewDecoder(resp.Body).Decode(&qualityProfilesReadResponse)
	if err != nil {
		return nil, fmt.Errorf("readQualityProfilesFromApi: Failed to decode json into struct: %+v", err)
	}

	return &qualityProfilesReadResponse, nil
}

func flattenReadQualityProfilesResponse(qualityProfiles []GetQualityProfile) []interface{} {
	qualityProfilesList := []interface{}{}

	for _, qualityProfile := range qualityProfiles {
		values := map[string]interface{}{
			"key":        qualityProfile.Key,
			"name":       qualityProfile.Name,
			"language":   qualityProfile.Language,
			"is_default": qualityProfile.IsDefault,
		}

		qualityProfilesList = append(qualityProfilesList, values)
	}

	return qualityProfilesList
}
