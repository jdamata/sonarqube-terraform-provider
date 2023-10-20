package sonarqube

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tidwall/gjson"
)

var sonarqubeProvider *schema.Provider

// Provider for sonarqube
func Provider() *schema.Provider {
	sonarqubeProvider = &schema.Provider{
		// Provider configuration
		Schema: map[string]*schema.Schema{
			"user": {
				Type:         schema.TypeString,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"SONAR_USER", "SONARQUBE_USER"}, nil),
				Optional:     true,
				RequiredWith: []string{"pass"},
			},
			"pass": {
				Type:         schema.TypeString,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"SONAR_PASS", "SONARQUBE_PASS"}, nil),
				Optional:     true,
				Sensitive:    true,
				RequiredWith: []string{"user"},
			},
			"token": {
				Type:         schema.TypeString,
				DefaultFunc:  schema.MultiEnvDefaultFunc([]string{"SONAR_TOKEN", "SONARQUBE_TOKEN"}, nil),
				Optional:     true,
				Sensitive:    true,
				ExactlyOneOf: []string{"pass"},
			},
			"host": {
				Type:        schema.TypeString,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"SONAR_HOST", "SONARQUBE_HOST"}, nil),
				Required:    true,
			},
			"http_proxy": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"installed_version": {
				Type:        schema.TypeString,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"INSTALLED_VERSION"}, ""),
				Optional:    true,
			},
			"installed_edition": {
				Type:        schema.TypeString,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"INSTALLED_EDITION"}, ""),
				Optional:    true,
			},
			"tls_insecure_skip_verify": {
				Optional:    true,
				Type:        schema.TypeBool,
				Description: "Allows ignoring insecure certificates when set to true. Defaults to false. Disabling TLS verification is dangerous and should only be done for local testing.",
				Default:     false,
			},
			"anonymize_user_on_delete": {
				Optional:    true,
				Type:        schema.TypeBool,
				Description: "Allows anonymizing users on destroy. Requires Sonarqube version >= 9.7.",
				Default:     false,
			},
		},
		// Add the resources supported by this provider to this map.
		ResourcesMap: map[string]*schema.Resource{
			"sonarqube_alm_azure":                          resourceSonarqubeAlmAzure(),
			"sonarqube_azure_binding":                      resourceSonarqubeAzureBinding(),
			"sonarqube_group":                              resourceSonarqubeGroup(),
			"sonarqube_group_member":                       resourceSonarqubeGroupMember(),
			"sonarqube_permission_template":                resourceSonarqubePermissionTemplate(),
			"sonarqube_permissions":                        resourceSonarqubePermissions(),
			"sonarqube_plugin":                             resourceSonarqubePlugin(),
			"sonarqube_project":                            resourceSonarqubeProject(),
			"sonarqube_project_main_branch":                resourceSonarqubeProjectMainBranch(),
			"sonarqube_portfolio":                          resourceSonarqubePortfolio(),
			"sonarqube_qualityprofile":                     resourceSonarqubeQualityProfile(),
			"sonarqube_qualityprofile_project_association": resourceSonarqubeQualityProfileProjectAssociation(),
			"sonarqube_qualitygate":                        resourceSonarqubeQualityGate(),
			"sonarqube_qualitygate_project_association":    resourceSonarqubeQualityGateProjectAssociation(),
			"sonarqube_qualitygate_usergroup_association":  resourceSonarqubeQualityGateUsergroupAssociation(),
			"sonarqube_user":                               resourceSonarqubeUser(),
			"sonarqube_user_external_identity":             resourceSonarqubeUserExternalIdentity(),
			"sonarqube_user_token":                         resourceSonarqubeUserToken(),
			"sonarqube_webhook":                            resourceSonarqubeWebhook(),
			"sonarqube_rule":                               resourceSonarqubeRule(),
			"sonarqube_setting":                            resourceSonarqubeSettings(),
			"sonarqube_qualityprofile_activate_rule":       resourceSonarqubeQualityProfileRule(),
			"sonarqube_alm_github":                         resourceSonarqubeAlmGithub(),
			"sonarqube_github_binding":                     resourceSonarqubeGithubBinding(),
			"sonarqube_alm_gitlab":                         resourceSonarqubeAlmGitlab(),
			"sonarqube_gitlab_binding":                     resourceSonarqubeGitlabBinding(),
			"sonarqube_new_code_periods":                   resourceSonarqubeNewCodePeriodsBinding(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"sonarqube_user":           dataSourceSonarqubeUser(),
			"sonarqube_group":          dataSourceSonarqubeGroup(),
			"sonarqube_project":        dataSourceSonarqubeProject(),
			"sonarqube_portfolio":      dataSourceSonarqubePortfolio(),
			"sonarqube_qualityprofile": dataSourceSonarqubeQualityProfile(),
			"sonarqube_qualitygate":    dataSourceSonarqubeQualityGate(),
			"sonarqube_rule":           dataSourceSonarqubeRule(),
		},
		ConfigureFunc: configureProvider,
	}
	return sonarqubeProvider
}

// ProviderConfiguration contains the sonarqube providers configuration
type ProviderConfiguration struct {
	httpClient              *retryablehttp.Client
	sonarQubeURL            url.URL
	sonarQubeVersion        *version.Version
	sonarQubeEdition        string
	sonarQubeAnonymizeUsers bool
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	transport := cleanhttp.DefaultPooledTransport()
	if proxy, ok := d.GetOk("http_proxy"); ok {
		proxyUrl, err := url.Parse(proxy.(string))
		if err != nil {
			return nil, fmt.Errorf("failed to parse http_proxy: %+v", err)
		}
		transport.Proxy = http.ProxyURL(proxyUrl)
	}
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: d.Get("tls_insecure_skip_verify").(bool),
	}

	client := retryablehttp.NewClient()
	client.HTTPClient.Transport = transport

	host, err := url.Parse(d.Get("host").(string))
	if err != nil {
		return nil, fmt.Errorf("failed to parse sonarqube host: %+v", err)
	}

	sonarQubeURL := url.URL{
		Scheme:     host.Scheme,
		Host:       host.Host,
		Path:       host.Path,
		ForceQuery: true,
	}

	if token, ok := d.GetOk("token"); ok {
		sonarQubeURL.User = url.UserPassword(token.(string), "")
	} else {
		sonarQubeURL.User = url.UserPassword(d.Get("user").(string), d.Get("pass").(string))
	}

	// If either of installed_version or installed_edition is not set, we need to fetch them from the API
	installedVersion := d.Get("installed_version").(string)
	installedEdition := d.Get("installed_edition").(string)
	if installedVersion == "" || installedEdition == "" {
		installedVersionAPI, installedEditionAPI, err := sonarqubeSystemInfo(client, sonarQubeURL)
		if err != nil {
			return nil, err
		}

		if installedVersion == "" {
			installedVersion = installedVersionAPI
		}
		if installedEdition == "" {
			installedEdition = installedEditionAPI
		}
	}

	parsedInstalledVersion, err := version.NewVersion(installedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sonarqube version to a version: %+v", err)
	}

	minimumVersion, _ := version.NewVersion("9.9")
	if parsedInstalledVersion.LessThan(minimumVersion) {
		return nil, fmt.Errorf("unsupported version of sonarqube. Minimum supported version is %+v. Running version is %+v", minimumVersion, installedVersion)
	}

	// Anonymizing users is supported since version 9.7. For older releases we reset it to false:
	minimumVersionForAnonymize, _ := version.NewVersion("9.7")
	anonymizeUsers := d.Get("anonymize_user_on_delete").(bool) && parsedInstalledVersion.GreaterThanOrEqual(minimumVersionForAnonymize)

	return &ProviderConfiguration{
		httpClient:              client,
		sonarQubeURL:            sonarQubeURL,
		sonarQubeVersion:        parsedInstalledVersion,
		sonarQubeEdition:        installedEdition,
		sonarQubeAnonymizeUsers: anonymizeUsers,
	}, nil
}

func sonarqubeSystemInfo(client *retryablehttp.Client, sonarqube url.URL) (string, string, error) {
	// Make request to sonarqube version endpoint
	sonarqube.Path = strings.TrimSuffix(sonarqube.Path, "/") + "/api/system/info"
	resp, err := httpRequestHelper(
		client,
		"GET",
		sonarqube,
		http.StatusOK,
		"sonarqubeHealth",
	)
	if err != nil {
		return "", "", fmt.Errorf("cannot get sonarqube version/edition. Please configure installed_version and installed_edition: %+v", err)
	}
	defer resp.Body.Close()

	// Read in the response
	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse response body on GET sonarqube system/info api: %+v", err)
	}

	sonarqubeVersion := gjson.GetBytes(responseData, "System.Version").String()
	sonarqubeEdition := gjson.GetBytes(responseData, "System.Edition").String()
	return sonarqubeVersion, sonarqubeEdition, nil
}
