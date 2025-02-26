package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_group_member", &resource.Sweeper{
		Name: "sonarqube_group_member",
		F:    testSweepSonarqubeGroupMemberSweeper,
	})
}

// TODO: implement sweeper to clean up groups: https://www.terraform.io/docs/extend/testing/acceptance-tests/sweepers.html
func testSweepSonarqubeGroupMemberSweeper(r string) error {
	return nil
}

func testAccSonarqubeGroupMemberBasicConfig(rnd string, groupName string, loginName string) string {
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
		`, rnd, groupName, loginName)
}

func TestAccSonarqubeGroupMemberBasic(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_group_member." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeGroupMemberBasicConfig(rnd, "testAccSonarqubeGroup", "testAccSonarqubeUser"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeGroup"),
					resource.TestCheckResourceAttr(name, "login_name", "testAccSonarqubeUser"),
				),
			},
		},
	})
}
