package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ReadLanguageResponse for unmarshalling response body of Quality Gate read
type ReadLanguageResponse struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// GetLanguages for unmarshalling response body of languages get
type GetLanguages struct {
	ID        string                 `json:"id"`
	Languages []ReadLanguageResponse `json:"languages"`
}

func dataSourceSonarqubeLanguages() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get Sonarqube language resources.",
		Read:        dataSourceSonarqubeLanguagesRead,
		Schema: map[string]*schema.Schema{
			"search": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Search languages by key or name.",
			},
			"languages": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The key of the languagee.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the language.",
						},
					},
				},
				Description: "The list of languages.",
			},
		},
	}
}

func dataSourceSonarqubeLanguagesRead(d *schema.ResourceData, m interface{}) error {
	d.SetId(fmt.Sprintf("%d", schema.HashString(d.Get("search"))))

	languagesReadResponse, err := readLanguagesFromApi(d, m)
	if err != nil {
		return err
	}

	d.Set("languages", flattenReadLanguagesResponse(languagesReadResponse.Languages))

	return nil
}

func readLanguagesFromApi(d *schema.ResourceData, m interface{}) (*GetLanguages, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/languages/list"

	if data, ok := d.GetOk("search"); ok {
		search := data.(string)
		sonarQubeURL.RawQuery = url.Values{
			"q": []string{search},
		}.Encode()
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"readLanguagesFromApi",
	)
	if err != nil {
		return nil, fmt.Errorf("readLanguagesFromApi: Failed to call api/languages/list: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	languagesReadResponse := GetLanguages{}
	err = json.NewDecoder(resp.Body).Decode(&languagesReadResponse)
	if err != nil {
		return nil, fmt.Errorf("resourceLanguagesRead: Failed to decode json into struct: %+v", err)
	}

	// Make sure the order is always the same for when we are comparing lists of languages
	sort.Slice(languagesReadResponse.Languages, func(i, j int) bool {
		return languagesReadResponse.Languages[i].Key < languagesReadResponse.Languages[j].Key
	})

	return &languagesReadResponse, nil
}

func flattenReadLanguagesResponse(languages []ReadLanguageResponse) []interface{} {
	languagesList := []interface{}{}

	for _, language := range languages {
		values := map[string]interface{}{
			"key":  language.Key,
			"name": language.Name,
		}

		languagesList = append(languagesList, values)
	}

	return languagesList
}
