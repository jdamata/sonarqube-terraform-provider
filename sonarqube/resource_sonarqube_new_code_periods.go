package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// NewCodePeriods for unmarshalling response body from new_code_periods list definitions.
type NewCodePeriod struct {
	Project        string `json:"projectKey"`
	Branch         string `json:"branchKey"`
	Type           string `json:"type"`
	Value          string `json:"value,omitempty"`
	EffectiveValue string `json:"effectiveValue"`
	Inherited      bool   `json:"inherited"`
}

// New Code Period types
type NewCodePeriodType string

const (
	SpecificAnalysis NewCodePeriodType = "SPECIFIC_ANALYSIS"
	PreviousVersion  NewCodePeriodType = "PREVIOUS_VERSION"
	NumberOfDays     NewCodePeriodType = "NUMBER_OF_DAYS"
	ReferenceBranch  NewCodePeriodType = "REFERENCE_BRANCH"
)

// Returns the resource represented by this file.
func resourceSonarqubeNewCodePeriodsBinding() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeNewCodePeriodsCreate,
		Read:   resourceSonarqubeNewCodePeriodsRead,
		Update: resourceSonarqubeNewCodePeriodsCreate,
		Delete: resourceSonarqubeNewCodePeriodsDelete,

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"branch": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				RequiredWith: []string{"project"},
			},
			"project": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"type": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{string(SpecificAnalysis), string(PreviousVersion), string(NumberOfDays), string(ReferenceBranch)}, false)),
			},
			"value": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceSonarqubeNewCodePeriodsCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/new_code_periods/set"

	periodType := NewCodePeriodType(d.Get("type").(string))
	rawQuery := url.Values{
		"type": []string{string(periodType)},
	}

	id := "newCodePeriod"

	branch := d.Get("branch").(string)
	project := d.Get("project").(string)
	value := d.Get("value").(string)

	if branch != "" {
		rawQuery.Add("branch", branch)
		id += "/" + branch

		rawQuery.Add("project", project)
		id += "/" + project
	} else if project != "" {
		rawQuery.Add("project", project)
		id += "/" + project
	}
	if value != "" {
		rawQuery.Add("value", value)
	}

	if periodType == PreviousVersion {
		if value != "" {
			return fmt.Errorf("resourceSonarqubeNewCodePeriodsCreate: 'value' must be unset when the 'type' is %s", periodType)
		}
	} else if value == "" {
		return fmt.Errorf("resourceSonarqubeNewCodePeriodsCreate: 'value' must be configured when the 'type' is %s", periodType)
	}

	if periodType == SpecificAnalysis && branch == "" {
		return fmt.Errorf("resourceSonarqubeNewCodePeriodsCreate: 'branch' must be configured when the 'type' is %s", periodType)
	} else if periodType == ReferenceBranch && branch == "" && project == "" {
		return fmt.Errorf("resourceSonarqubeNewCodePeriodsCreate: both 'branch' and 'project' must be configured when the 'type' is %s", periodType)
	} else if periodType == NumberOfDays && !regexp.MustCompile(`^\d+$`).MatchString(value) {
		return fmt.Errorf("resourceSonarqubeNewCodePeriodsCreate: 'value' must be a numeric string when the 'type' is %s", periodType)
	}

	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeNewCodePeriodsCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d.SetId(id)

	return resourceSonarqubeNewCodePeriodsRead(d, m)
}

func resourceSonarqubeNewCodePeriodsRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/new_code_periods/show"

	rawQuery := url.Values{}
	branch := d.Get("branch").(string)
	if branch != "" {
		rawQuery.Add("branch", branch)
	}
	project := d.Get("project").(string)
	if project != "" {
		rawQuery.Add("project", project)
	}
	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeNewCodePeriodsRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	NewCodePeriodsReadResponse := NewCodePeriod{}
	err = json.NewDecoder(resp.Body).Decode(&NewCodePeriodsReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeNewCodePeriodsRead: Failed to decode json into struct: %+v", err)
	}
	// Check that the project and branch match
	if branch == NewCodePeriodsReadResponse.Branch && project == NewCodePeriodsReadResponse.Project {
		id := "newCodePeriod"
		if NewCodePeriodsReadResponse.Branch != "" {
			id += "/" + NewCodePeriodsReadResponse.Branch
		}
		if NewCodePeriodsReadResponse.Project != "" {
			id += "/" + NewCodePeriodsReadResponse.Project
		}
		d.SetId(id)
		return nil
	}

	return fmt.Errorf("resourceSonarqubeNewCodePeriodsRead: Failed to find new code period: %+v", d.Id())
}

func resourceSonarqubeNewCodePeriodsDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/new_code_periods/unset"

	rawQuery := url.Values{}
	branch := d.Get("branch").(string)
	if branch != "" {
		rawQuery.Add("branch", branch)
	}
	project := d.Get("project").(string)
	if project != "" {
		rawQuery.Add("project", project)
	}
	sonarQubeURL.RawQuery = rawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubeNewCodePeriodsDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
