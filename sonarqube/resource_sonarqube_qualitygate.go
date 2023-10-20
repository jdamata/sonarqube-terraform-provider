package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ReadQualityGateConditionsResponse for unmarshalling response body of Quality Gate read
type ReadQualityGateConditionsResponse struct {
	ID     string `json:"id"`
	Metric string `json:"metric"`
	OP     string `json:"op"`
	Error  string `json:"error"`
}

// GetQualityGate for unmarshalling response body of quality gate get
type GetQualityGate struct {
	ID         string                              `json:"id"`
	Name       string                              `json:"name"`
	Conditions []ReadQualityGateConditionsResponse `json:"conditions"`
	IsBuiltIn  bool                                `json:"isBuiltIn"`
	Actions    QualityGateActions                  `json:"actions"`
}

// QualityGateActions used in GetQualityGate
type QualityGateActions struct {
	Rename            bool `json:"rename"`
	SetAsDefault      bool `json:"setAsDefault"`
	Copy              bool `json:"copy"`
	AssociateProjects bool `json:"associateProjects"`
	Delete            bool `json:"delete"`
	ManageConditions  bool `json:"manageConditions"`
}

// CreateQualityGateResponse for unmarshalling response body of quality gate creation
type CreateQualityGateResponse struct {
	Name string `json:"name"`
}

// Returns the resource represented by this file.
func resourceSonarqubeQualityGate() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeQualityGateCreate,
		Read:   resourceSonarqubeQualityGateRead,
		Update: resourceSonarqubeQualityGateUpdate,
		Delete: resourceSonarqubeQualityGateDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeQualityGateImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"copy_from": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"condition"},
			},
			"is_default": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "When set to true this Quality Gate is set as default",
				Default:     false,
			},
			"condition": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "A list of conditions that the gate uses",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"metric": {
							Type:     schema.TypeString,
							Required: true,
						},
						"op": {
							Type:     schema.TypeString,
							Required: true,
						},
						"threshold": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceSonarqubeQualityGateCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL

	copying_gate := false
	if gate_to_copy, ok := d.GetOk("copy_from"); ok {
		copying_gate = true
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/copy"
		sonarQubeURL.RawQuery = url.Values{
			"name":       []string{d.Get("name").(string)},
			"sourceName": []string{gate_to_copy.(string)},
		}.Encode()
	} else {
		if _, ok := d.GetOk("condition"); !ok {
			return fmt.Errorf("resourceQualityGateCreate: either copy_from or at least one condition block must be specified for a quality gate")
		}
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/create"
		sonarQubeURL.RawQuery = url.Values{
			"name": []string{d.Get("name").(string)},
		}.Encode()
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceQualityGateCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	qualityGateResponse := CreateQualityGateResponse{}
	err = json.NewDecoder(resp.Body).Decode(&qualityGateResponse)
	if err != nil {
		return fmt.Errorf("resourceQualityGateCreate: Failed to decode json into struct: %+v", err)
	}

	d.SetId(qualityGateResponse.Name)

	qualityGateReadResponse, err := readQualityGateFromApi(d, m)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityGateCreate: Failed to read the quality gate from the API: %+v", err)
	}

	// SonarQube 9.9 and above will automatically create "Clean as you code" conditions for new quality gates
	// If we are not copying a gate then we need to synchronise the conditions from the newly created gate with
	// the ones declared on the terraform resource
	if !copying_gate {
		changes, err := synchronizeConditions(d, m, &qualityGateReadResponse.Conditions)
		if err != nil {
			return fmt.Errorf("resourceSonarqubeQualityGateCreate: Failed to synchronise quality gate conditions: %+v", err)
		}

		// If we did make any changes then re-read the quality gate from the API.
		if changes {
			qualityGateReadResponse, err = readQualityGateFromApi(d, m)
			if err != nil {
				return fmt.Errorf("resourceSonarqubeQualityGateCreate: Failed to read the quality gate after conditions were updated: %+v", err)
			}
		}
	}

	if d.Get("is_default").(bool) {
		err := setDefaultQualityGate(d, m, true)
		if err != nil {
			return fmt.Errorf("resourceSonarqubeQualityGateCreate: Failed to set this quality gate as default: %+v", err)
		}
	}

	updateResourceDataFromQualityGateReadResponse(d, qualityGateReadResponse)
	return nil
}

func resourceSonarqubeQualityGateRead(d *schema.ResourceData, m interface{}) error {
	qualityGateReadResponse, err := readQualityGateFromApi(d, m)
	if err != nil {
		return err
	}
	updateResourceDataFromQualityGateReadResponse(d, qualityGateReadResponse)
	// Api returns if true if set as default is available. when is_default=true setAsDefault=false so is_default=true
	d.Set("is_default", !qualityGateReadResponse.Actions.SetAsDefault)
	return nil
}

var lock_update_default sync.Mutex

func resourceSonarqubeQualityGateUpdate(d *schema.ResourceData, m interface{}) error {
	_, copied_gate := d.GetOk("copy_from")

	if _, has_conditions := d.GetOk("condition"); !(copied_gate || has_conditions) {
		return fmt.Errorf("resourceQualityGateCreate: either copy_from or at least one condition block must be specified for a quality gate")
	}

	if d.HasChange("name") {
		err := updateQualityGateName(d, m)
		if err != nil {
			return fmt.Errorf("resourceSonarqubeQualityGateUpdate: Failed to change the name of the quality gate: %+v", err)
		}
		d.SetId(d.Get("name").(string))
	}

	qualityGateReadResponse, err := readQualityGateFromApi(d, m)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeQualityGateUpdate: Failed to read the quality gate from the API: %+v", err)
	}

	conditionsChanged := false

	// We only need to update the conditions if this is not a copied gate - they will still exist from when it was created originally
	if !copied_gate {
		conditionsChanged, err = synchronizeConditions(d, m, &qualityGateReadResponse.Conditions)
		if err != nil {
			return fmt.Errorf("resourceSonarqubeQualityGateUpdate: Failed to synchronise quality gate conditions: %+v", err)
		}
	}

	// If we are changing the default then we need to ensure this next section is synchronous in case another
	// quality gate is being set as the default in a parallel thread.
	lock_update_default.Lock()
	defer lock_update_default.Unlock()

	defaultChanged := d.HasChange("is_default")

	// If we made any condition changes or want to change the default quality gate then re-read the quality gate from the API.
	if conditionsChanged || defaultChanged {
		qualityGateReadResponse, err = readQualityGateFromApi(d, m)
		if err != nil {
			return fmt.Errorf("resourceSonarqubeQualityGateUpdate: Failed to read the quality gate after conditions were updated: %+v", err)
		}
	}

	if defaultChanged {
		newDefault := d.Get("is_default").(bool)
		// If we are changing this to NOT be the default but it is already NOT the default (e.g. because some other quality gate has been
		// explicitly set as default) then we don't need to do anything (and accidentally set Sonar way as default!)
		// In all other cases where the default has changed, we do need to update it.
		if newDefault != !qualityGateReadResponse.Actions.SetAsDefault {
			err := setDefaultQualityGate(d, m, newDefault)
			if err != nil {
				return fmt.Errorf("resourceSonarqubeQualityGateUpdate: Failed to set this quality gate as default: %+v", err)
			}
		}
	}

	updateResourceDataFromQualityGateReadResponse(d, qualityGateReadResponse)
	return nil
}

func resourceSonarqubeQualityGateDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/destroy"

	sonarQubeURL.RawQuery = url.Values{
		"name": []string{d.Id()},
	}.Encode()

	// If this is the default quality gate then we need to default it back to "Sonar way" so there is still a default
	if d.Get("is_default").(bool) {
		err := setDefaultQualityGate(d, m, false)
		if err != nil {
			return err
		}
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceQualityGateDelete",
	)
	if err != nil {
		return fmt.Errorf("resourceQualityGateDelete: Failed to delete quality gate: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeQualityGateImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if err := resourceSonarqubeQualityGateRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func setDefaultQualityGate(d *schema.ResourceData, m interface{}, setDefault bool) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/set_as_default"

	if setDefault {
		sonarQubeURL.RawQuery = url.Values{
			"name": []string{d.Get("name").(string)},
		}.Encode()
	} else {
		sonarQubeURL.RawQuery = url.Values{
			"name": []string{"Sonar way"},
		}.Encode()
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"setDefaultQualityGate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func readQualityGateFromApi(d *schema.ResourceData, m interface{}) (*GetQualityGate, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/show"

	sonarQubeURL.RawQuery = url.Values{
		"name": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"readQualityGateFromApi",
	)
	if err != nil {
		return nil, fmt.Errorf("readQualityGateFromApi: Failed to call api/qualitygates/show: %+v", err)
	}
	defer resp.Body.Close()

	// Decode response into struct
	qualityGateReadResponse := GetQualityGate{}
	err = json.NewDecoder(resp.Body).Decode(&qualityGateReadResponse)
	if err != nil {
		return nil, fmt.Errorf("resourceQualityGateRead: Failed to decode json into struct: %+v", err)
	}

	// Make sure the order is always the same for when we are comparing lists of conditions
	sort.Slice(qualityGateReadResponse.Conditions, func(i, j int) bool {
		return qualityGateReadResponse.Conditions[i].Metric < qualityGateReadResponse.Conditions[j].Metric
	})

	return &qualityGateReadResponse, nil
}

func synchronizeConditions(d *schema.ResourceData, m interface{}, apiQualityGateConditions *[]ReadQualityGateConditionsResponse) (bool, error) {

	changed := false
	qualityGateConditions := d.Get("condition").([]interface{})

	// Make sure the order is always the same for when we are comparing lists of conditions
	sort.Slice(qualityGateConditions, func(i, j int) bool {
		return qualityGateConditions[i].(map[string]interface{})["metric"].(string) < qualityGateConditions[j].(map[string]interface{})["metric"].(string)
	})

	// Determine which conditions have been added or changed and update those
	for i, condition := range qualityGateConditions {
		conditionId, err := addOrUpdateCondition(d, m, apiQualityGateConditions, condition, &changed)
		if err != nil {
			return changed, err
		}
		if conditionId != "" {
			qualityGateConditions[i].(map[string]interface{})["id"] = conditionId
		}
	}

	// Determine if any conditions have been removed and delete them
	err := removeDeletedConditions(apiQualityGateConditions, qualityGateConditions, m, &changed)
	if err != nil {
		return changed, err
	}

	if changed {
		d.Set("condition", qualityGateConditions)
	}

	return changed, nil
}

func addOrUpdateCondition(d *schema.ResourceData, m interface{}, apiQualityGateConditions *[]ReadQualityGateConditionsResponse, condition interface{}, changed *bool) (string, error) {
	metric := condition.(map[string]interface{})["metric"].(string)
	op := condition.(map[string]interface{})["op"].(string)
	threshold := condition.(map[string]interface{})["threshold"].(string)

	// Update the condition if it already exists
	for _, apiCondition := range *apiQualityGateConditions {
		if metric == apiCondition.Metric {
			if op != apiCondition.OP || threshold != apiCondition.Error {
				err := updateCondition(apiCondition.ID, metric, op, threshold, m)
				if err != nil {
					return "", fmt.Errorf("addOrUpdateCondition: Failed to update condition '%s': %+v", metric, err)
				}
				*changed = true
			}
			return "", nil
		}
	}

	// Add the condition because it does not already exist
	conditionId, err := createCondition(d.Id(), metric, op, threshold, m)
	if err != nil {
		return conditionId, fmt.Errorf("addOrUpdateCondition: Failed to create condition '%s': %+v", metric, err)
	}
	*changed = true
	return conditionId, nil
}

func removeDeletedConditions(apiQualityGateConditions *[]ReadQualityGateConditionsResponse, qualityGateConditions []interface{}, m interface{}, changed *bool) error {
	for _, apiCondition := range *apiQualityGateConditions {
		found := false

		for _, newCondition := range qualityGateConditions {
			if newCondition.(map[string]interface{})["metric"] == apiCondition.Metric {
				found = true
				break
			}
		}

		if !found {
			err := deleteCondition(apiCondition.ID, m)
			if err != nil {
				return fmt.Errorf("removeDeletedConditions: Failed to delete condition '%s': %+v", apiCondition.Metric, err)
			}
			*changed = true
		}
	}
	return nil
}

func updateResourceDataFromQualityGateReadResponse(d *schema.ResourceData, qualityGateReadResponse *GetQualityGate) {
	d.SetId(qualityGateReadResponse.Name)
	d.Set("name", qualityGateReadResponse.Name)
	// Copied gates do not have condition blocks so we don't want to populate from the API.
	if _, copiedGate := d.GetOk("copy_from"); !copiedGate {
		d.Set("condition", flattenReadQualityGateConditionsResponse(&qualityGateReadResponse.Conditions))
	}
}

func createCondition(qualityGateName string, metric string, op string, threshold string, m interface{}) (string, error) {

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/create_condition"

	sonarQubeURL.RawQuery = url.Values{
		"gateName": []string{qualityGateName},
		"metric":   []string{metric},
		"op":       []string{op},
		"error":    []string{threshold},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"createCondition",
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Decode response into struct
	qualityGateConditionResponse := ReadQualityGateConditionsResponse{}
	err = json.NewDecoder(resp.Body).Decode(&qualityGateConditionResponse)
	if err != nil {
		return "", fmt.Errorf("createCondition: Failed to decode json into struct: %+v", err)
	}

	return qualityGateConditionResponse.ID, nil
}

func updateCondition(id, metric, op, threshold string, m interface{}) error {

	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/update_condition"

	sonarQubeURL.RawQuery = url.Values{
		"id":     []string{id},
		"metric": []string{metric},
		"op":     []string{op},
		"error":  []string{threshold},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"updateCondition",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func deleteCondition(id string, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/delete_condition"
	sonarQubeURL.RawQuery = url.Values{
		"id": []string{id},
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

func updateQualityGateName(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/rename"

	old, new := d.GetChange("name")
	sonarQubeURL.RawQuery = url.Values{
		"currentName": []string{old.(string)},
		"name":        []string{new.(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"updateQualityGateName",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func flattenReadQualityGateConditionsResponse(input *[]ReadQualityGateConditionsResponse) []interface{} {
	if input == nil || len(*input) == 0 {
		return make([]interface{}, 0)
	}

	flatConditions := make([]interface{}, len(*input))

	for i, condition := range *input {
		c := make(map[string]interface{})

		c["id"] = condition.ID
		c["metric"] = condition.Metric
		c["op"] = condition.OP
		c["threshold"] = condition.Error

		flatConditions[i] = c
	}

	return flatConditions
}
