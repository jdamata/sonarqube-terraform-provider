package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubeUsersDataSourceConfig(rnd string, name string, email string, password string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "%[2]s"
			email      = "%[3]s"
			password   = "%[4]s"
		}
		
		data "sonarqube_users" "%[1]s" {
			search = sonarqube_user.%[1]s.name
		}`, rnd, name, email, password)
}

func TestAccSonarqubeUsersDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_users." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUsersDataSourceConfig(rnd, "testAccSonarqubeUsersDataSource", "terraform-test@sonarqube.com", "secret-Sauce37!"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "users.#", "1"),
					resource.TestCheckResourceAttr(name, "users.0.login_name", "testAccSonarqubeUsersDataSource"),
					resource.TestCheckResourceAttr(name, "users.0.name", "testAccSonarqubeUsersDataSource"),
					resource.TestCheckResourceAttr(name, "users.0.email", "terraform-test@sonarqube.com"),
					resource.TestCheckResourceAttr(name, "users.0.is_local", "true"),
				),
			},
		},
	})
}
