package sonarqube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/exp/slices"
)

// Portfolio used in Portfolio
type Portfolio struct {
	Key              string             `json:"key"`
	Name             string             `json:"name"`
	Desc             string             `json:"desc,omitempty"`
	Qualifier        string             `json:"qualifier"`
	Visibility       string             `json:"visibility"`
	SelectionMode    string             `json:"selectionMode"`
	Branch           string             `json:"branch,omitempty"`
	Tags             []string           `json:"tags,omitempty"`
	Regexp           string             `json:"regexp,omitempty"`
	SelectedProjects []PortfolioProject `json:"selectedProjects,omitempty"`
}

// Portfolio project
type PortfolioProject struct {
	ProjectKey       string   `json:"projectKey"`
	SelectedBranches []string `json:"selectedBranches,omitempty"`
}

const (
	NONE   = "NONE"
	MANUAL = "MANUAL"
	TAGS   = "TAGS"
	REGEXP = "REGEXP"
	REST   = "REST"
)

// Returns the resource represented by this file.
func resourceSonarqubePortfolio() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubePortfolioCreate,
		Read:   resourceSonarqubePortfolioRead,
		Update: resourceSonarqubePortfolioUpdate,
		Delete: resourceSonarqubePortfolioDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubePortfolioImport,
		},
		// Validation that runs after the read in plan has completed (https://developer.hashicorp.com/terraform/plugin/sdkv2/resources/customizing-differences)
        CustomizeDiff: customdiff.All(
			func(_ context.Context, d *schema.ResourceDiff, meta interface{}) error {
				return validatePortfolioResource(d)
			},
       ),

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"qualifier": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"visibility": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "public",
				ForceNew:     true, // TODO: There currently isn't an API to update this in-place, even though it's possible in the UI
				ValidateFunc: validation.StringInSlice([]string{"public", "private"}, false),
			},
			"selection_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      NONE,
				ForceNew:     false,
				ValidateFunc: validation.StringInSlice([]string{NONE, MANUAL, TAGS, REGEXP, REST}, false),
				
			},
			"branch": { // Only active for TAGS, REGEXP and REST
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    false,
				Description: "Which branch to analyze. If nothing, or '' is specified, the main branch is used.",
			},
			"tags": { // Only active for TAGS
				Type:          schema.TypeList,
				Optional:      true,
				ForceNew:      false,
				ConflictsWith: []string{"regexp", "selected_projects"},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"regexp": { // Only active for REGEXP
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      false,
				ConflictsWith: []string{"selected_projects", "tags"},
				ValidateFunc:  validation.StringIsValidRegExp,
			},
			"selected_projects": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      false,
				ConflictsWith: []string{"tags", "regexp"},
				Description:   "A set of projects to add to the portfolio.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"project_key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The project key of the project to add to the portfolio",
						},
						"selected_branches": {
							Type:        schema.TypeSet,
							Optional:    true,
							Description: "A set of branches for the project to add to the portfolio",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func checkPortfolioSupport(conf *ProviderConfiguration) error {
	if strings.ToLower(conf.sonarQubeEdition) != "enterprise" {
		return fmt.Errorf("portfolios are only supported in the Enterprise edition of SonarQube. You are using: SonarQube %s version %s", conf.sonarQubeEdition, conf.sonarQubeVersion)
	}
	return nil
}

// Validate the selection_mode and its corresponding fields
func validatePortfolioResource(d *schema.ResourceDiff) error {
	switch selectionMode := d.Get("selection_mode"); selectionMode {
	case NONE, REST:
		return nil

	case MANUAL:
		selectedProjects := d.Get("selected_projects").(*schema.Set).List()
		if len(selectedProjects) == 0 {
			return fmt.Errorf("validatePortfolioResource: When selection_mode is set to MANUAL, you need atleast 1 selected_project, got: %+v", selectedProjects)
		}
		return nil

	case TAGS:
		tags := d.Get("tags").([]interface{})
		if len(tags) == 0 {
			return fmt.Errorf("validatePortfolioResource: When selection_mode is set to TAGS, you need atleast 1 tag, got: %+v", d.Get("tags"))
		}

		for _, tag := range d.Get("tags").([]interface{}) {
			tagString := fmt.Sprint(tag)
			if len(tagString) == 0 {
				return fmt.Errorf("validatePortfolioResource: When selection_mode is set to TAGS, each tag must be non 0, got: %s", tagString)
			}
		}
		return nil

	case REGEXP:
		regexp := d.Get("regexp").(string)
		if len(regexp) == 0 {
			return fmt.Errorf("validatePortfolioResource: When selection_mode is set to REGEXP, regexp must be set, got: \"%s\"", regexp)
		}
		return nil

	default:
		return fmt.Errorf("resourceSonarqubePortfolioCreate: selection_mode needs to be set to one of NONE, MANUAL, TAGS, REGEXP, REST")
	}

}

func portfolioSetSelectionMode(d *schema.ResourceData, m interface{}, sonarQubeURL url.URL) error {
	var endpoint string
	switch selectionMode := d.Get("selection_mode"); selectionMode {
	case NONE:
		endpoint = "/api/views/set_none_mode"
		sonarQubeURL.RawQuery = url.Values{
			"portfolio": []string{d.Get("key").(string)},
		}.Encode()

	case MANUAL:
		endpoint = "/api/views/set_manual_mode"
		sonarQubeURL.RawQuery = url.Values{
			"portfolio": []string{d.Get("key").(string)},
		}.Encode()

	case TAGS:
		endpoint = "/api/views/set_tags_mode"

		var tags []string
		for _, v := range d.Get("tags").([]interface{}) {
			tags = append(tags, fmt.Sprint(v))
		}
		tagsCSV := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(tags)), ","), "[]")

		urlParameters := url.Values{
			"portfolio": []string{d.Get("key").(string)},
			"tags":      []string{tagsCSV},
		}

		// SonarQube handles "" like it actually is a name of a branch, see PR for reference: https://github.com/jdamata/terraform-provider-sonarqube/pull/150
		branch := d.Get("branch").(string)
		if len(branch) > 0 {
			urlParameters.Add("branch", branch)
		}

		sonarQubeURL.RawQuery = urlParameters.Encode()

	case REGEXP:
		endpoint = "/api/views/set_regexp_mode"

		urlParameters := url.Values{
			"portfolio": []string{d.Get("key").(string)},
			"regexp":    []string{d.Get("regexp").(string)},
		}

		// SonarQube handles "" like it actually is a name of a branch, see PR for reference: https://github.com/jdamata/terraform-provider-sonarqube/pull/150
		branch := d.Get("branch").(string)
		if len(branch) > 0 {
			urlParameters.Add("branch", branch)
		}

		sonarQubeURL.RawQuery = urlParameters.Encode()

	case REST:
		endpoint = "/api/views/set_remaining_projects_mode"

		urlParameters := url.Values{
			"portfolio": []string{d.Get("key").(string)},
		}

		// SonarQube handles "" like it actually is a name of a branch, see PR for reference: https://github.com/jdamata/terraform-provider-sonarqube/pull/150
		branch := d.Get("branch").(string)
		if len(branch) > 0 {
			urlParameters.Add("branch", branch)
		}

		sonarQubeURL.RawQuery = urlParameters.Encode()

	default:
		return fmt.Errorf("resourceSonarqubePortfolioCreate: selection_mode needs to be set to one of NONE, MANUAL, TAGS, REGEXP, REST")
	}

	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + endpoint

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubePortfolioCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// The rest of the options populate the portfolio in the "setMode" call. MANUAL portfolios needs to be manually populated afterwards
	if selectionMode := d.Get("selection_mode").(string); selectionMode == MANUAL {
		portfolioReadResponse, err := readPortfolioFromApi(d, m)
		if err != nil {
			return fmt.Errorf("resourceSonarqubePortfolioCreate: Failed to read the portfolio from the API: %+v", err)
		}

		err = synchronizeSelectedProjects(d, m, &portfolioReadResponse.SelectedProjects)
		if err != nil {
			return fmt.Errorf("resourceSonarqubePortfolioCreate: Failed to synchronise portfolio projects: %+v", err)
		}
	}

	return nil
}

func resourceSonarqubePortfolioCreate(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/create"

	sonarQubeURL.RawQuery = url.Values{
		"description": []string{d.Get("description").(string)},
		"key":         []string{d.Get("key").(string)},
		"name":        []string{d.Get("name").(string)},
		"visibility":  []string{d.Get("visibility").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceSonarqubePortfolioCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	portfolioResponse := Portfolio{}
	err = json.NewDecoder(resp.Body).Decode(&portfolioResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubePortfolioCreate: Failed to decode json into struct: %+v", err)
	}

	d.SetId(portfolioResponse.Key)

	err = portfolioSetSelectionMode(d, m, m.(*ProviderConfiguration).sonarQubeURL)
	if err != nil {
		return err
	}

	return resourceSonarqubePortfolioRead(d, m)
}

func resourceSonarqubePortfolioRead(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	portfolioReadResponse, err := readPortfolioFromApi(d, m)
	if err != nil {
		return err
	}
	updateResourceDataFromPortfolioReadResponse(d, portfolioReadResponse)
	return nil
}

func resourceSonarqubePortfolioUpdate(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	if d.HasChanges("name", "description") {
		sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/update"
		sonarQubeURL.RawQuery = url.Values{
			"key":         []string{d.Id()},
			"description": []string{d.Get("description").(string)},
			"name":        []string{d.Get("name").(string)},
		}.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL,
			http.StatusOK,
			"resourceSonarqubePortfolioUpdate",
		)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube Portfolio Name and Description: %+v", err)
		}
		defer resp.Body.Close()
	}

	if d.HasChanges("selection_mode", "branch", "tags", "regexp", "selected_projects") {
		err := portfolioSetSelectionMode(d, m, m.(*ProviderConfiguration).sonarQubeURL)
		if err != nil {
			return fmt.Errorf("error updating Sonarqube selection mode: %+v", err)
		}
	}

	return resourceSonarqubePortfolioRead(d, m)
}

func resourceSonarqubePortfolioDelete(d *schema.ResourceData, m interface{}) error {
	if err := checkPortfolioSupport(m.(*ProviderConfiguration)); err != nil {
		return err
	}

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/delete"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,

		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceSonarqubePortfolioDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubePortfolioImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubePortfolioRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func updateResourceDataFromPortfolioReadResponse(d *schema.ResourceData, portfolioReadResponse *Portfolio) {

	d.SetId(portfolioReadResponse.Key)
	d.Set("key", portfolioReadResponse.Key)
	d.Set("name", portfolioReadResponse.Name)
	d.Set("description", portfolioReadResponse.Desc)
	d.Set("qualifier", portfolioReadResponse.Qualifier)
	d.Set("visibility", portfolioReadResponse.Visibility)
	d.Set("selection_mode", portfolioReadResponse.SelectionMode)

	// These fields may or may not be set in the reposnse from SonarQube
	if len(portfolioReadResponse.Tags) > 0 {
		d.Set("tags", portfolioReadResponse.Tags)
	}
	if len(portfolioReadResponse.Branch) > 0 {
		d.Set("branch", portfolioReadResponse.Branch)
	}
	if len(portfolioReadResponse.Regexp) > 0 {
		d.Set("regexp", portfolioReadResponse.Regexp)
	}

	if len(portfolioReadResponse.SelectedProjects) > 0 {
		d.Set("selected_projects", flattenReadPortfolioSelectedProjectsResponse(&portfolioReadResponse.SelectedProjects))
	}

}

func readPortfolioFromApi(d *schema.ResourceData, m interface{}) (*Portfolio, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/show"

	sonarQubeURL.RawQuery = url.Values{
		"key": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"readPortfolioFromApi",
	)
	if err != nil {
		return nil, fmt.Errorf("readPortfolioFromApi: Failed to call api/views/show: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	portfolioReadResponse := Portfolio{}
	err = json.NewDecoder(resp.Body).Decode(&portfolioReadResponse)
	if err != nil {
		return nil, fmt.Errorf("readPortfolioFromApi: Failed to decode json into struct: %+v", err)
	}

	// Make sure the order is always the same for when we are comparing lists of conditions
	sort.Slice(portfolioReadResponse.SelectedProjects, func(i, j int) bool {
		return portfolioReadResponse.SelectedProjects[i].ProjectKey < portfolioReadResponse.SelectedProjects[j].ProjectKey
	})

	return &portfolioReadResponse, nil
}

func synchronizeSelectedProjects(d *schema.ResourceData, m interface{}, apiPortfolioSelectedProjects *[]PortfolioProject) error {
	portfolioSelectedProjects := d.Get("selected_projects").(*schema.Set).List()

	// Make sure the order is always the same for when we are comparing lists of projects
	sort.Slice(portfolioSelectedProjects, func(i, j int) bool {
		return portfolioSelectedProjects[i].(map[string]interface{})["project_key"].(string) < portfolioSelectedProjects[j].(map[string]interface{})["project_key"].(string)
	})

	// Determine which conditions have been added or changed and update those
	for _, project := range portfolioSelectedProjects {
		err := addOrUpdateSelectedProject(d, m, apiPortfolioSelectedProjects, project)
		if err != nil {
			return err
		}
	}

	// Determine if any conditions have been removed and delete them
	portfolioKey := d.Get("key").(string)
	err := removeDeletedSelectedProject(portfolioKey, apiPortfolioSelectedProjects, portfolioSelectedProjects, m)
	if err != nil {
		return err
	}

	return nil
}

func addOrUpdateSelectedProject(d *schema.ResourceData, m interface{}, apiPortfolioSelectedProjects *[]PortfolioProject, project interface{}) error {
	portfolioKey := d.Get("key").(string)
	projectKey := project.(map[string]interface{})["project_key"].(string)

	selectedBranches := make([]string, 0)
	for _, v := range project.(map[string]interface{})["selected_branches"].(*schema.Set).List() {
		if v != nil {
			selectedBranches = append(selectedBranches, v.(string))
		}
	}

	// Update the project if it already exists and the selected branches has changed, otherwise do nothing
	for _, apiProject := range *apiPortfolioSelectedProjects {
		if projectKey == apiProject.ProjectKey {
			if !stringSlicesEqual(selectedBranches, apiProject.SelectedBranches, true) {
				err := updateSelectedProject(portfolioKey, projectKey, selectedBranches, apiProject.SelectedBranches, m)
				if err != nil {
					return fmt.Errorf("addOrUpdateSelectedProject: Failed to update project '%s': %+v", projectKey, err)
				}
				return nil
			}
			return nil
		}
	}

	// Add the project because it does not already exist
	err := addSelectedProject(portfolioKey, projectKey, selectedBranches, m)
	if err != nil {
		return fmt.Errorf("addOrUpdateCondition: Failed to add project '%s': %+v", projectKey, err)
	}
	return nil
}

func addSelectedProject(portfolioKey, projectKey string, selectedBranches []string, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/add_project"

	sonarQubeURL.RawQuery = url.Values{
		"key":     []string{portfolioKey},
		"project": []string{projectKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent, // For some reason this endpoint returns 204 on success...
		"addSelectedProject",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	for _, branch := range selectedBranches {
		addSelectedProjectBranch(portfolioKey, projectKey, branch, m)
	}


	return nil
}

func updateSelectedProject(portfolioKey, projectKey string, selectedBranches, apiSelectedBranches []string, m interface{}) error {
	// For each branch in the terraform schema, make sure they are also in SonarQube
	for _, branch := range selectedBranches {
		if !slices.Contains(apiSelectedBranches, branch) {
			addSelectedProjectBranch(portfolioKey, projectKey, branch, m)
		}
	}

	// For each branch in SonarQube, ensure it exists in the terraform schema, otherwise remove it
	for _, branch := range apiSelectedBranches {
		if !slices.Contains(selectedBranches, branch) {
			deleteSelectedProjectBranch(portfolioKey, projectKey, branch, m)
		}
	}

	return nil
}

func addSelectedProjectBranch(portfolioKey, projectKey, branch string, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/add_project_branch"

	sonarQubeURL.RawQuery = url.Values{
		"key":     []string{portfolioKey},
		"project": []string{projectKey},
		"branch":  []string{branch},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent, // For some reason this endpoint returns 204 on success...
		"addSelectedProject",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func deleteSelectedProjectBranch(portfolioKey, projectKey, branch string, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/remove_project_branch"

	sonarQubeURL.RawQuery = url.Values{
		"key":     []string{portfolioKey},
		"project": []string{projectKey},
		"branch":  []string{branch},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent, // For some reason this endpoint returns 204 on success...
		"addSelectedProject",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func removeDeletedSelectedProject(portfolioKey string, apiPortfolioSelectedProjects *[]PortfolioProject, portfolioSelectedProjects []interface{}, m interface{}) error {
	for _, apiProject := range *apiPortfolioSelectedProjects {
		found := false
		for _, project := range portfolioSelectedProjects {
			if project.(map[string]interface{})["project_key"] == apiProject.ProjectKey {
				found = true
				break
			}
		}
		if !found {
			err := deleteSelectedProject(portfolioKey, apiProject.ProjectKey, m)
			if err != nil {
				return fmt.Errorf("removeDeletedSelectedProject: Failed to delete project from portfolio '%s': %+v", apiProject.ProjectKey, err)
			}
		}
	}
	return nil
}

func deleteSelectedProject(portfolioKey, projectKey string, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/views/remove_project"
	sonarQubeURL.RawQuery = url.Values{
		"key":     []string{portfolioKey},
		"project": []string{projectKey},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"deleteCondition",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func flattenReadPortfolioSelectedProjectsResponse(input *[]PortfolioProject) []interface{} {
	if input == nil || len(*input) == 0 {
		return make([]interface{}, 0)
	}

	flatSelectedProjects := make([]interface{}, len(*input))

	for i, project := range *input {
		p := make(map[string]interface{})

		p["project_key"] = project.ProjectKey
		p["selected_branches"] = project.SelectedBranches

		flatSelectedProjects[i] = p
	}

	return flatSelectedProjects
}
