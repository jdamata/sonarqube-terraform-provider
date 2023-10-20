package sonarqube

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type Webhook struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Url    string `json:"url"`
	Secret string `json:"secret"`
}

type CreateWebhookResponse struct {
	Webhook *Webhook `json:"webhook"`
}

type ListWebhooksResponse struct {
	Webhooks []*Webhook `json:"webhooks"`
}

// Returns the resource represented by this file.
func resourceSonarqubeWebhook() *schema.Resource {
	return &schema.Resource{
		Create: resourceSonarqubeWebhookCreate,
		Read:   resourceSonarqubeWebhookRead,
		Update: resourceSonarqubeWebhookUpdate,
		Delete: resourceSonarqubeWebhookDelete,
		Importer: &schema.ResourceImporter{
			State: resourceSonarqubeWebhookImport,
		},

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"secret": {
				Type:      schema.TypeString,
				Sensitive: true,
				Optional:  true,
				Computed:  true,
			},
			"project": {
				Type:        schema.TypeString,
				Description: "The key of the project that will own the webhook.",
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceSonarqubeWebhookCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/webhooks/create"

	params := url.Values{
		"name": []string{d.Get("name").(string)},
		"url":  []string{d.Get("url").(string)},
	}
	if secret, ok := d.GetOk("secret"); ok {
		params.Set("secret", secret.(string))
	}
	if project, ok := d.GetOk("project"); ok {
		params.Set("project", project.(string))
	}

	sonarQubeURL.RawQuery = params.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusOK,
		"resourceWebhookCreate",
	)
	if err != nil {
		return fmt.Errorf("resourceWebhookCreate: Failed to call %s: %+v", sonarQubeURL.Path, err)
	}
	defer resp.Body.Close()

	webhookResponse := CreateWebhookResponse{}
	err = json.NewDecoder(resp.Body).Decode(&webhookResponse)
	if err != nil {
		return fmt.Errorf("resourceWebhookCreate: Failed to decode json into struct: %+v", err)
	}

	d.SetId(webhookResponse.Webhook.Key)

	return resourceSonarqubeWebhookRead(d, m)
}

// unfortunately, there doesn't seem to be a way to get a webhook by its ID. the best we can do is list all webhooks and
// loop through the result until we find the one we're looking for.
func resourceSonarqubeWebhookRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/webhooks/list"

	if project, ok := d.GetOk("project"); ok {
		rawQuery := url.Values{
			"project": []string{string(project.(string))},
		}
		sonarQubeURL.RawQuery = rawQuery.Encode()
	}

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL,
		http.StatusOK,
		"resourceWebhookRead",
	)
	if err != nil {
		return fmt.Errorf("resourceWebhookRead: Failed to call %s: %+v", sonarQubeURL.Path, err)
	}
	defer resp.Body.Close()

	webhookResponse := ListWebhooksResponse{}
	err = json.NewDecoder(resp.Body).Decode(&webhookResponse)
	if err != nil {
		return fmt.Errorf("resourceWebhookRead: Failed to decode json into struct: %+v", err)
	}

	for _, webhook := range webhookResponse.Webhooks {
		log.Printf("[DEBUG][resourceSonarqubeWebhookRead] webhook.Key: '%s' vs %s ", webhook.Key, d.Id())
		if webhook.Key == d.Id() {
			d.Set("name", webhook.Name)
			d.Set("url", webhook.Url)
			// Field 'project' is not included in the webhook response object, so it is imported from the parameter.
			if project, ok := d.GetOk("project"); ok {
				d.Set("project", project.(string))
			}
			// Version 10.1 of sonarqube does not return the secret in the api response anymore. Field 'secret' replaced by flag 'hasSecret' in response
			// Instead we just set the secret in state to the value being passed in to avoid constant drifts
			if secret, ok := d.GetOk("secret"); ok {
				d.Set("secret", secret.(string))
			}
			return nil
		}
	}

	return fmt.Errorf("resourceWebhookRead: Failed to find webhook with key %s", d.Id())
}

func resourceSonarqubeWebhookUpdate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/webhooks/update"

	params := url.Values{
		"webhook": []string{d.Id()},
		"name":    []string{d.Get("name").(string)},
		"url":     []string{d.Get("url").(string)},
	}
	project := d.Get("project").(string)
	if project != "" {
		params.Set("project", project)
	}
	if secret, ok := d.GetOk("secret"); ok {
		params.Set("secret", secret.(string))
	}
	sonarQubeURL.RawQuery = params.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceWebhookUpdate",
	)
	if err != nil {
		return fmt.Errorf("resourceWebhookUpdate: Failed to update webhook: %+v", err)
	}
	defer resp.Body.Close()

	return resourceSonarqubeWebhookRead(d, m)
}

func resourceSonarqubeWebhookDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/webhooks/delete"

	sonarQubeURL.RawQuery = url.Values{
		"webhook": []string{d.Id()},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL,
		http.StatusNoContent,
		"resourceWebhookDelete",
	)
	if err != nil {
		return fmt.Errorf("resourceWebhookDelete: Failed to delete webhook: %+v", err)
	}
	defer resp.Body.Close()

	return nil
}

func resourceSonarqubeWebhookImport(d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	// import id in format {key}/{project}
	importIdComponents := strings.SplitN(d.Id(), "/", 2)

	if len(importIdComponents) == 2 {
		log.Printf("[DEBUG][resourceSonarqubeWebhookImport] Import id: '%+v' is in format {key/project:%s/%s}", d.Id(), importIdComponents[0], importIdComponents[1])
		d.Set("project", importIdComponents[1])
	} else if len(importIdComponents) == 1 {
		log.Printf("[DEBUG][resourceSonarqubeWebhookImport] Import id: '%+v' is in format {key:%s}", d.Id(), importIdComponents[0])
	} else {
		return nil, fmt.Errorf("resourceSonarqubeWebhookImport: Import id: '%+v' is not in format {key}/{project} or {key}", d.Id())
	}

	// set Id to key for Read
	d.SetId(importIdComponents[0])
	if err := resourceSonarqubeWebhookRead(d, m); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
