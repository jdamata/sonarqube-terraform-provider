package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// GetTokens struct
type GetTokens struct {
	Login  string  `json:"login,omitempty"`
	Tokens []Token `json:"userTokens,omitempty"`
}

// Token struct
type Token struct {
	Login          string       `json:"login,omitempty"`
	Name           string       `json:"name,omitempty"`
	Token          string       `json:"token,omitempty"`
	ExpirationDate string       `json:"expirationDate,omitempty"`
	Type           string       `json:"type,omitempty"`
	CreatedAt      string       `json:"createdAt,omitempty"`
	IsExpired      bool         `json:"isExpired,omitempty"`
	Project        TokenProject `json:"project,omitempty"`
}

type TokenProject struct {
	Key  string `json:"key,omitempty"`
	Name string `json:"name,omitempty"`
}

// Token types
type TokenType string

const (
	UserToken            TokenType = "USER_TOKEN"
	GlobalAnalysisToken  TokenType = "GLOBAL_ANALYSIS_TOKEN"
	ProjectAnalysisToken TokenType = "PROJECT_ANALYSIS_TOKEN"
)

// Returns the resource represented by this file.
func resourceSonarqubeUserToken() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeUserTokenCreate,
		Read:   resourceSonarqubeUserTokenRead,
		Delete: resourceSonarqubeUserTokenDelete,

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 100)),
			},
			"login_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"expiration_date": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"token": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"type": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          UserToken,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{string(UserToken), string(GlobalAnalysisToken), string(ProjectAnalysisToken)}, false)),
			},
			"project_key": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSonarqubeUserTokenCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_tokens/generate"

	tokenType := TokenType(d.Get("type").(string))
	rawQuery := url.Values{
		"name": []string{d.Get("name").(string)},
		"type": []string{string(tokenType)},
	}

	if tokenType == UserToken {
		loginName := d.Get("login_name").(string)
		if loginName != "" {
			rawQuery.Add("login", loginName)
		}
	} else if tokenType == ProjectAnalysisToken {
		projectKey := d.Get("project_key").(string)
		if projectKey == "" {
			return fmt.Errorf("resourceSonarqubeUserTokenCreate: 'project_key' must be configured when the token 'type' is %s", ProjectAnalysisToken)
		}
		rawQuery.Add("projectKey", projectKey)
	}

	if _, ok := d.GetOk("expiration_date"); ok {
		rawQuery.Add("expirationDate", d.Get("expiration_date").(string))
	}

	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeUserTokenCreate",
	)
	if err != nil {
		return fmt.Errorf("error creating Sonarqube user token: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	tokenResponse := Token{}
	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeUserTokenCreate: Failed to decode json into struct: %+v", err)
	}

	if tokenResponse.Login != "" {
		// the ID consists of the login_name and the token name (foo/bar)
		d.SetId(fmt.Sprintf("%s/%s", d.Get("login_name").(string), d.Get("name").(string)))
		// we set the token value here as the API wont return it later
		if tokenResponse.Token != "" {
			d.Set("token", tokenResponse.Token)
		} else {
			return fmt.Errorf("resourceSonarqubeUserTokenCreate: Create response didn't contain the token")
		}
	} else {
		return fmt.Errorf("resourceSonarqubeUserTokenCreate: Create response didn't contain the user login")
	}

	return resourceSonarqubeUserTokenRead(d, m)
}

func resourceSonarqubeUserTokenRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_tokens/search"

	// split d.Id into login_name and the token name (foo/bar)
	login := strings.Split(d.Id(), "/")
	if login[0] != "" {
		sonarQubeURL.RawQuery = url.Values{
			"login": []string{login[0]},
		}.Encode()
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeUserTokenRead",
	)
	if err != nil {
		return fmt.Errorf("error reading Sonarqube user tokens: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	getTokensResponse := GetTokens{}
	err = json.NewDecoder(resp.Body).Decode(&getTokensResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeUserTokenCreate: Failed to decode json into struct: %+v", err)
	}

	// Loop over all user token to see if the current token exists.
	if getTokensResponse.Tokens != nil {
		for _, value := range getTokensResponse.Tokens {
			if d.Get("name").(string) == value.Name {
				d.SetId(fmt.Sprintf("%s/%s", d.Get("login_name").(string), d.Get("name").(string)))
				d.Set("login_name", getTokensResponse.Login)
				d.Set("name", value.Name)
				if value.ExpirationDate != "" {
					dateReceived, errTimeParse := time.Parse("2006-01-02T15:04:05-0700", value.ExpirationDate)
					if errTimeParse != nil {
						return fmt.Errorf("resourceSonarqubeUserTokenCreate: Failed to parse ExpirationDate: %+v", err)
					}
					d.Set("expiration_date", dateReceived.Format("2006-01-02"))
				}
				return nil
			}
		}
	}

	return fmt.Errorf("resourceSonarqubeUserTokenCreate: Failed to find user token: %+v", d.Id())
}

func resourceSonarqubeUserTokenDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/user_tokens/revoke"
	rawQuery := url.Values{
		"name": []string{d.Get("name").(string)},
	}
	login := d.Get("login_name").(string)
	if login != "" {
		rawQuery.Add("login", login)
	}

	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeUserTokenDelete",
	)
	if err != nil {
		return fmt.Errorf("error deleting Sonarqube user token: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}
