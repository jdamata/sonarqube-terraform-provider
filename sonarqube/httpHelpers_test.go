package sonarqube

import (
	"errors"
	"reflect"
	"testing"
)

// TestCensorError calls censorError with sample log messages , checking
// for a valid return value.
func TestCensorError(t *testing.T) {
	cases := []struct {
		errorMessage error
		token        string
		expected     string
	}{
		{
			errorMessage: errors.New("Error updating SonarQube user: failed to execute http request: POST https://PASSWORD:@sonarqube.example.com/api/users/update_identity_provider?login=gitlab-john-doe&newExternalIdentity=john-doe&newExternalProvider=gitlab giving up after 5 attempt(s). Request: &{0xab1940 0xc00021c600}"),
			token:        "PASSWORD",
			expected:     "Error updating SonarQube user: failed to execute http request: POST https://********:@sonarqube.example.com/api/users/update_identity_provider?login=gitlab-john-doe&newExternalIdentity=john-doe&newExternalProvider=gitlab giving up after 5 attempt(s). Request: &{0xab1940 0xc00021c600}",
		},
	}
	for _, c := range cases {
		out := censorError(c.errorMessage, c.token)
		if !reflect.DeepEqual(out.Error(), c.expected) {
			t.Fatalf("Error matching output and expected: %#v vs %#v", out.Error(), c.expected)
		}
	}
}
