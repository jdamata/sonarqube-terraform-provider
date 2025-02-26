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

func dataSourceSonarqubeQualityGates() *schema.Resource {
	return &schema.Resource{
		Description: "Use this data source to get Sonarqube quality gates resources",
		Read:        dataSourceSonarqubeQualityGatesRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Search quality gates by name.",
			},
			"ignore_missing": {
				Type:        schema.TypeBool,
				Default:     false,
				Optional:    true,
				Description: "If set to true, the data source will not fail if the quality gate does not exist.",
			},
			"quality_gates": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The id of the Quality Gate.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the Quality Gate.",
						},
						"copy_from": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Origin of the Quality Gate",
						},
						"is_default": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Quality Gate default.",
						},
						"condition": {
							Type:     schema.TypeList,
							Computed: true,
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
							Description: "List of Quality Gate conditions.",
						},
					},
				},
				Description: "The list of quality gates.",
			},
		},
	}
}

func dataSourceSonarqubeQualityGatesRead(d *schema.ResourceData, m interface{}) error {
	d.SetId(fmt.Sprintf("%d", schema.HashString(d.Get("name"))))

	qualityGateReadResponse, err := readQualityGatesFromApi(d, m)
	if err != nil {
		return err
	}

	if qualityGateReadResponse != nil {
		d.Set("quality_gates", flattenReadQualityGateResponse(*qualityGateReadResponse))
	} else {
		d.Set("quality_gates", []interface{}{})
	}

	return nil
}

func readQualityGatesFromApi(d *schema.ResourceData, m interface{}) (*GetQualityGate, error) {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/qualitygates/show"

	RawQuery := url.Values{}

	if name, ok := d.GetOk("name"); ok {
		RawQuery.Add("name", name.(string))
	}

	sonarQubeURL.RawQuery = RawQuery.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"readQualityGatesFromApi",
	)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound && d.Get("ignore_missing").(bool) {
			// If the quality gate does not exist, we don't want to fail the data source
			return nil, nil
		}
		return nil, fmt.Errorf("error reading Sonarqube quality gates: %+v", err)
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

func flattenReadQualityGateResponse(qualityGate GetQualityGate) []interface{} {
	qualityGatesList := []interface{}{}

	values := map[string]interface{}{
		"id":   qualityGate.Name,
		"name": qualityGate.Name,
		// Api returns if true if set as default is available. when is_default=true setAsDefault=false so is_default=true
		"is_default": !qualityGate.Actions.SetAsDefault,
		"condition":  flattenReadQualityGateConditionsResponse(&qualityGate.Conditions),
	}

	qualityGatesList = append(qualityGatesList, values)

	return qualityGatesList
}
