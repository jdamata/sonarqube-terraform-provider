package sonarqube

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

// ErrorResponse struct
type ErrorResponse struct {
	Errors []ErrorMessage `json:"errors,omitempty"`
}

// ErrorMessage struct
type ErrorMessage struct {
	Message string `json:"msg,omitempty"`
}

// Paging used in /search API endpoints
type Paging struct {
	PageIndex int64 `json:"pageIndex"`
	PageSize  int64 `json:"pageSize"`
	Total     int64 `json:"total"`
}

// helper function to make api request to sonarqube
func httpRequestHelper(client *retryablehttp.Client, method string, sonarqubeURL url.URL, expectedResponseCode int, errormsg string) (http.Response, error) {
	// Prepare request
	req, err := retryablehttp.NewRequest(method, sonarqubeURL.String(), http.NoBody)
	if err != nil {
		return http.Response{}, fmt.Errorf("failed to prepare http request: %v", censorError(err, sonarqubeURL.User.String()))
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return http.Response{}, fmt.Errorf("failed to execute http request: %v", censorError(err, sonarqubeURL.User.String()))
	}

	// Check response code
	if resp.StatusCode != expectedResponseCode {
		if resp.Body == http.NoBody {
			// No error message in the body
			return *resp, fmt.Errorf("statusCode: %v does not match expectedResponseCode: %v", resp.StatusCode, expectedResponseCode)
		}

		// The response body has content, try to decode the error message
		errorResponse := ErrorResponse{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		if err != nil {
			return *resp, fmt.Errorf("failed to decode error response json into struct: %+v", err)
		}
		return *resp, fmt.Errorf("API returned an error: %+v", errorResponse.Errors[0].Message)
	}

	return *resp, nil
}

// https://github.com/jdamata/terraform-provider-sonarqube/issues/201
// go-retryablehttp error contains the token/user:pass in plaintext.
// We want to censor that secret before logging the error
func censorError(err error, secret string) error {
	// Replace the secret with asterisks of the same length
	asterisks := strings.Repeat("*", len(secret))
	result := strings.Replace(err.Error(), secret, asterisks, -1)
	return errors.New(result)
}
