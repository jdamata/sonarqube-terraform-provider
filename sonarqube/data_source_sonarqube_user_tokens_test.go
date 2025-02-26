package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubeUserTokensDataSourceConfig(rnd string, name string) string {
	return fmt.Sprintf(`
		 resource "sonarqube_user" "%[1]s" {
            login_name = "%[2]s"
            name       = "%[2]s"
            password   = "secret-Sauce37!"
        }
        resource "sonarqube_user_token" "%[1]s" {
            login_name = sonarqube_user.%[1]s.login_name
            name       = "%[2]s"
        }
		
		data "sonarqube_user_tokens" "%[1]s" {
			login_name = sonarqube_user.%[1]s.login_name
			depends_on = [sonarqube_user_token.%[1]s]
		}`, rnd, name)
}

func TestAccSonarqubeUserTokensDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_user_tokens." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserTokensDataSourceConfig(rnd, "testAccSonarqubeUserTokensDataSource"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "user_tokens.#", "1"),
					resource.TestCheckResourceAttr(name, "user_tokens.0.name", "testAccSonarqubeUserTokensDataSource"),
				),
			},
		},
	})
}
