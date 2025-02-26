package sonarqube

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_user_external_identity", &resource.Sweeper{
		Name: "sonarqube_user_external_identity",
		F:    testSweepSonarqubeUserExteernalIdentitySweeper,
	})
}

func testSweepSonarqubeUserExteernalIdentitySweeper(r string) error {
	return nil
}

func testAccSonarqubeUserExternalIdentityConfig(rnd string, login string, externalIdentity string, externalProvider string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "Test User"
			email      = "terraform-test@sonarqube.com"
			is_local   = false
		}

		resource "sonarqube_user_external_identity" "%[1]s" {
			login_name         = sonarqube_user.%[1]s.login_name
			external_identity  = "%[3]s"
			external_provider  = "%[4]s"
		}`, rnd, login, externalIdentity, externalProvider)
}

func TestAccSonarqubeUserExternalIdentity(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_user_external_identity." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserExternalIdentityConfig(rnd, "testAccSonarqubeUser", "terraform-test@sonarqube.com", "sonarqube"), // Provider "sonarbube" is deprecated in 9.8. "LDAP" works in 9.8 but not in current LTS.
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "login_name", "testAccSonarqubeUser"),
					resource.TestCheckResourceAttr(name, "external_identity", "terraform-test@sonarqube.com"),
					resource.TestCheckResourceAttr(name, "external_provider", "sonarqube"),
				),
			},
		},
	})
}

func testAccSonarqubeUserExternalIdentityLocalUserConfig(rnd string, login string, externalIdentity string, externalProvider string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "Test User"
			email      = "terraform-test@sonarqube.com"
			is_local   = true
			password   = "secret-Sauce1"
		}

		resource "sonarqube_user_external_identity" "%[1]s" {
			login_name         = sonarqube_user.%[1]s.login_name
			external_identity  = "%[3]s"
			external_provider  = "%[4]s"
		}`, rnd, login, externalIdentity, externalProvider)
}

func TestAccSonarqubeUserExternalLocalUserIdentity(t *testing.T) {
	rnd := generateRandomResourceName()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      testAccSonarqubeUserExternalIdentityLocalUserConfig(rnd, "testAccSonarqubeUserExternalLocalUserIdentity", "terraform-test@sonarqube.com", "sonarqube"), // Provider "sonarbube" is deprecated in 9.8. "LDAP" works in 9.8 but not in current LTS.
				ExpectError: regexp.MustCompile("Error setting external identity: Sonarqube user 'testAccSonarqubeUserExternalLocalUserIdentity' is not 'external'"),
			},
		},
	})
}
