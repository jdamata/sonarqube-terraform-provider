package sonarqube

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_user_token", &resource.Sweeper{
		Name: "sonarqube_user_token",
		F:    testSweepSonarqubeUserTokenSweeper,
	})
}

func testSweepSonarqubeUserTokenSweeper(r string) error {
	return nil
}

func testAccSonarqubeUserTokenBasicConfig(rnd string, name string) string {
	return fmt.Sprintf(`
        resource "sonarqube_user" "%[1]s" {
            login_name = "%[2]s"
            name       = "%[2]s"
            password   = "secret-Sauce37!"
        }
        resource "sonarqube_user_token" "%[1]s" {
            login_name = sonarqube_user.%[1]s.login_name
            name       = "%[2]s"
        }`, rnd, name)
}

func TestAccSonarqubeUserTokenBasic(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_user_token." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserTokenBasicConfig(rnd, "testAccSonarqubeUserToken"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserToken"),
				),
			},
		},
	})
}

func testAccSonarqubeUserTokenExpirationDateConfig(rnd string, name string, expiration_date string) string {
	return fmt.Sprintf(`
        resource "sonarqube_user" "%[1]s" {
            login_name = "%[2]s"
            name       = "%[2]s"
            password   = "secret-Sauce37!"
        }
        resource "sonarqube_user_token" "%[1]s" {
            login_name      = sonarqube_user.%[1]s.login_name
            name            = "%[2]s"
            expiration_date = "%s"
        }`, rnd, name, expiration_date)
}

func TestAccSonarqubeUserTokenWithExpirationDate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_user_token." + rnd
	expiration_date := time.Now().AddDate(0, 0, 5).Format("2006-01-02")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserTokenExpirationDateConfig(rnd, "testAccSonarqubeUserTokenWithExpirationDate", expiration_date),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserTokenWithExpirationDate"),
					resource.TestCheckResourceAttr(name, "expiration_date", expiration_date),
				),
			},
		},
	})
}

func testAccSonarqubeUserTokenNoLoginConfig(rnd string, name string) string {
	return fmt.Sprintf(`
        resource "sonarqube_user" "%[1]s" {
            login_name = "%[2]s"
            name       = "%[2]s"
            password   = "secret-Sauce37!"
        }
        resource "sonarqube_user_token" "%[1]s" {
            name       = "%[2]s"
        }`, rnd, name)
}

func testAccSonarqubeUserTokenNoLogin(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_user_token." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserTokenNoLoginConfig(rnd, "testAccSonarqubeUserTokenNoLogin"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserTokenNoLogin"),
					resource.TestCheckResourceAttr(name, "login", name),
				),
			},
		},
	})
}

func testAccSonarqubeUserTokenGlobalAnalysisTokenConfig(rnd string, name string) string {
	return fmt.Sprintf(`
        resource "sonarqube_user" "%[1]s" {
            login_name = "%[2]s"
            name       = "%[2]s"
            password   = "secret-Sauce37!"
        }
        resource "sonarqube_user_token" "%[1]s" {
            name       = "%[2]s"
            type       = "GLOBAL_ANALYSIS_TOKEN"
        }`, rnd, name)
}

func testAccSonarqubeUserTokenGlobalAnalysisToken(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_user_token." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserTokenGlobalAnalysisTokenConfig(rnd, "testAccSonarqubeUserTokenGlobalAnalysisToken"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserTokenGlobalAnalysisToken"),
					resource.TestCheckResourceAttr(name, "type", "GLOBAL_ANALYSIS_TOKEN"),
				),
			},
		},
	})
}

func testAccSonarqubeUserTokenProjectAnalysisTokenConfig(rnd string, name string) string {
	return fmt.Sprintf(`
        resource "sonarqube_user" "%[1]s" {
            login_name = "%[2]s"
            name       = "%[2]s"
            password   = "secret-Sauce37!"
        }
        resource "sonarqube_user_token" "%[1]s" {
            name        = "%[2]s"
            type        = "PROJECT_ANALYSIS_TOKEN"
            project_key = "my-project"
        }`, rnd, name)
}

func testAccSonarqubeUserTokenProjectAnalysisToken(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_user_token." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserTokenProjectAnalysisTokenConfig(rnd, "testAccSonarqubeUserTokenProjectAnalysisToken"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserTokenProjectAnalysisToken"),
					resource.TestCheckResourceAttr(name, "type", "PROJECT_ANALYSIS_TOKEN"),
				),
			},
		},
	})
}
