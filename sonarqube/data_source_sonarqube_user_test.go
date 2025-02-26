package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubeUserDataSourceConfig(rnd string, name string, email string, password string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "%[2]s"
			email      = "%[3]s"
			password   = "%[4]s"
		}
		
		data "sonarqube_user" "%[1]s" {
			login_name = sonarqube_user.%[1]s.login_name
		}`, rnd, name, email, password)
}

func TestAccSonarqubeUserDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_user." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserDataSourceConfig(rnd, "testAccSonarqubeUserDataSource", "terraform-test-user-data-source@sonarqube.com", "secret-Sauce37!"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "login_name", "testAccSonarqubeUserDataSource"),
					resource.TestCheckResourceAttr(name, "email", "terraform-test-user-data-source@sonarqube.com"),
				),
			},
		},
	})
}
