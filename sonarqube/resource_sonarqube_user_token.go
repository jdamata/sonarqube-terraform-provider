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
		Description: "Provides a Sonarqube User token resource. This can be used to manage Sonarqube User tokens.",
		Create:      resourceSonarqubeUserTokenCreate,
		Read:        resourceSonarqubeUserTokenRead,
		Delete:      resourceSonarqubeUserTokenDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeUserTokenImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 100)),
				Description:      "The name of the Token to create. Changing this forces a new resource to be created.",
			},
			"login_name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The login name of the User for which the token should be created. If not set, the token is created for the authenticated user. Changing this forces a new resource to be created.",
			},
			"expiration_date": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "The expiration date of the token being generated, in ISO 8601 format (YYYY-MM-DD). If not set, default to no expiration.",
			},
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The token value.",
			},
			"type": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          UserToken,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{string(UserToken), string(GlobalAnalysisToken), string(ProjectAnalysisToken)}, false)),
				Description:      "The kind of Token to create. Changing this forces a new resource to be created. Possible values are USER_TOKEN, GLOBAL_ANALYSIS_TOKEN, or PROJECT_ANALYSIS_TOKEN. Defaults to USER_TOKEN. If set to PROJECT_ANALYSIS_TOKEN, then the project_key must also be specified.",
			},
			"project_key": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The key of the only project that can be analyzed by the PROJECT_ANALYSIS TOKEN being created. Changing this forces a new resource to be created.",
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
		sonarQubeURL.String(),
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
	login_name := strings.Split(d.Id(), "/")
	if login_name[0] != "" {
		sonarQubeURL.RawQuery = url.Values{
			"login": []string{login_name[0]},
		}.Encode()
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
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
		return fmt.Errorf("resourceSonarqubeUserTokenRead: Failed to decode json into struct: %+v", err)
	}

	// Loop over all user token to see if the current token exists.
	if getTokensResponse.Tokens != nil {
		for _, value := range getTokensResponse.Tokens {
			if login_name[1] == value.Name {
				d.SetId(fmt.Sprintf("%s/%s", getTokensResponse.Login, value.Name))
				d.Set("login_name", getTokensResponse.Login)
				d.Set("name", value.Name)
				if value.ExpirationDate != "" {
					dateReceived, errTimeParse := time.Parse("2006-01-02T15:04:05-0700", value.ExpirationDate)
					if errTimeParse != nil {
						return fmt.Errorf("resourceSonarqubeUserTokenRead: Failed to parse ExpirationDate: %+v", errTimeParse)
					}
					d.Set("expiration_date", dateReceived.Format("2006-01-02"))
				}
				return nil
			}
		}
	}

	return fmt.Errorf("resourceSonarqubeUserTokenRead: Failed to find user token: %+v", d.Id())
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
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeUserTokenDelete",
	)
	if err != nil {
		return fmt.Errorf("error deleting Sonarqube user token: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeUserTokenImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubeUserTokenRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
