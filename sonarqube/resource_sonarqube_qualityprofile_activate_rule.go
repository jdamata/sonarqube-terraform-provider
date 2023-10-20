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

type Actives struct {
	QProfile string   `json:"qProfile"`
	Inherit  string   `json:"inherit"`
	Severity string   `json:"severity"`
	Params   []Params `json:"params"`
}

type GetActiveRules struct {
	Rule    Rule      `json:"rule"`
	Actives []Actives `json:"actives"`
}

func resourceSonarqubeQualityProfileRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeQualityProfileRuleCreate,
		Delete: resourceSonarqubeQualityProfileRuleDelete,
		Read:   resourceSonarqubeQualityProfileRuleRead,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeQualityProfileRuleImporter,
		},

		Schema: map[string]*schema.Schema{
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Quality Profile key. Can be obtained through api/qualityprofiles/search",
			},
			"params": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Parameters as semi-colon list of =, for example 'params=key1=v1;key2=v2' (Only for custom rule)",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"reset": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Reset severity and parameters of activated rule. Set the values defined on parent profile or from rule default values.",
				Default:     "false",
				ValidateDiagFunc: validation.ToDiagFunc(
					validation.StringInSlice(
						[]string{"true", "false", "yes", "no"},
						false,
					),
				),
			},
			"rule": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Rule key",
			},
			"severity": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Rule severity",
				ForceNew:    true,
				ValidateDiagFunc: validation.ToDiagFunc(
					validation.StringInSlice(
						[]string{"INFO", "MINOR", "MAJOR", "CRITICAL", "BLOCKER"},
						false,
					),
				),
			},
		},
	}
}

func resourceSonarqubeQualityProfileRuleCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/activate_rule"

	sonarQubeURL.RawQuery = url.Values{
		"key":      []string{d.Get("key").(string)},
		"params":   []string{d.Get("params").(string)},
		"reset":    []string{d.Get("reset").(string)},
		"rule":     []string{d.Get("rule").(string)},
		"severity": []string{d.Get("severity").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeQualityProfileRuleCreate",
	)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileRuleCreate: Failed to delete quality profile: %+v", err)
	}
	defer resp.Body.Close()

	d.SetId(d.Get("rule").(string))
	return resourceSonarqubeQualityProfileRuleRead(d, m)
}

func resourceSonarqubeQualityProfileRuleDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualityprofiles/deactivate_rule"
	sonarQubeURL.RawQuery = url.Values{
		"key":  []string{d.Get("key").(string)},
		"rule": []string{d.Get("rule").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubeQualityProfileRuleDelete",
	)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileRuleDelete: Failed to delete quality profile: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeQualityProfileRuleRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/rules/show"
	sonarQubeURL.RawQuery = url.Values{
		"key":     []string{d.Id()},
		"actives": []string{"true"},
	}.Encode()
	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeQualityProfileRuleRead",
	)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	activeRuleReadResponse := GetActiveRules{}
	err = json.NewDecoder(resp.Body).Decode(&activeRuleReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityProfileRuleRead: Failed to decode json into struct: %+v", err)
	}

	if d.Id() == activeRuleReadResponse.Rule.RuleKey {
		d.SetId(activeRuleReadResponse.Rule.RuleKey)
		return nil
	}

	return fmt.Errorf("resourceSonarqubeQualityProfileRuleRead: Failed to find project: %+v", d.Id())
}

func resourceSonarqubeQualityProfileRuleImporter(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubeQualityProfileRuleRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
