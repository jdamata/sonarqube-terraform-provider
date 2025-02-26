package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSonarqubeUserTokens() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get Sonarqube user token resources",
		Read:        dataSourceSonarqubeUserTokensRead,
		Schema: map[string]*schema.Schema{
			"login_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Search user tokens for the specified login name. Otherwise, tokens for the current user are listed. This login must exist and be active.",
			},
			"ignore_missing": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: "If set to true, the data source will not fail if the user does not exist.",
			},
			"user_tokens": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The id of the user token.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the user token.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the user token.",
						},
						"created_at": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The creation date of the user token.",
						},
						"expiration_date": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional:    true,
							Description: "The expiration date of the user token.",
						},
						"project_key": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional:    true,
							Description: "The key of the only project that can be analyzed by the user token.",
						},
					},
				},
				Description: "The list of user tokens.",
			},
		},
	}
}

func dataSourceSonarqubeUserTokensRead(d *schema.ResourceData, m interface{}) error {
	d.SetId(fmt.Sprintf("%d", schema.HashString(d.Get("login_name"))))

	userTokensReadResponse, err := readUserTokensFromApi(d, m)
	if err != nil {
		return err
	}

	if userTokensReadResponse != nil {
		userTokens, err := flattenReadUserTokensResponse(userTokensReadResponse.Login, userTokensReadResponse.Tokens)
		if err != nil {
			return err
		}

		d.Set("user_tokens", userTokens)
	} else {
		d.Set("user_tokens", []interface{}{})
	}

	return nil
}

func readUserTokensFromApi(d *schema.ResourceData, m interface{}) (*GetTokens, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_tokens/search"

	RawQuery := url.Values{}

	if login, ok := d.GetOk("login_name"); ok {
		RawQuery.Add("login", login.(string))
	}

	sonarQubeURL.RawQuery = RawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"readUserTokensFromApi",
	)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound && d.Get("ignore_missing").(bool) {
			// If the user does not exist, we don't want to fail the data source
			return nil, nil
		}
		return nil, fmt.Errorf("error reading Sonarqube user tokens: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	userTokensReadResponse := GetTokens{}
	err = json.NewDecoder(resp.Body).Decode(&userTokensReadResponse)
	if err != nil {
		return nil, fmt.Errorf("readUserTokensFromApi: Failed to decode json into struct: %+v", err)
	}

	return &userTokensReadResponse, nil
}

func flattenReadUserTokensResponse(login string, tokens []Token) ([]interface{}, error) {
	userTokensList := []interface{}{}

	for _, token := range tokens {
		values := map[string]interface{}{
			"id":   fmt.Sprintf("%s/%s", login, token.Name),
			"name": token.Name,
			"type": token.Type,
		}

		if token.Project.Key != "" {
			values["project_key"] = token.Project.Key
		}

		if token.CreatedAt != "" {
			date, err := time.Parse("2006-01-02T15:04:05-0700", token.CreatedAt)
			if err != nil {
				return nil, fmt.Errorf("flattenReadUserTokensResponse: Failed to parse CreatedAt: %+v", err)
			}
			values["created_at"] = date.Format("2006-01-02")
		}

		if token.ExpirationDate != "" {
			date, err := time.Parse("2006-01-02T15:04:05-0700", token.ExpirationDate)
			if err != nil {
				return nil, fmt.Errorf("flattenReadUserTokensResponse: Failed to parse ExpirationDate: %+v", err)
			}
			values["expiration_date"] = date.Format("2006-01-02")
		}

		userTokensList = append(userTokensList, values)
	}

	return userTokensList, nil
}
