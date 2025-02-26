package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubeGroupMembersDataSourceConfig(rnd string, groupName string, loginName string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s_user" {
			login_name = "%[3]s"
			name       = "Test User"
			email      = "terraform-test@sonarqube.com"
			password   = "secret-Sauce!"
		}

		resource "sonarqube_group" "%[1]s_group" {
			name        = "%[2]s"
		}

		resource "sonarqube_group_member" "%[1]s" {
			name       = sonarqube_group.%[1]s_group.name
			login_name = sonarqube_user.%[1]s_user.login_name
		}
		
		data "sonarqube_group_members" "%[1]s" {
			group = sonarqube_group.%[1]s_group.name
			depends_on = [sonarqube_group_member.%[1]s]
		}`, rnd, groupName, loginName)
}

func TestAccSonarqubeGroupMembersDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_group_members." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeGroupMembersDataSourceConfig(rnd, "testAccSonarqubeGroup", "testAccSonarqubeUser"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "members.#", "1"),
					resource.TestCheckResourceAttr(name, "members.0.login_name", "testAccSonarqubeUser"),
				),
			},
		},
	})
}
