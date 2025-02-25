package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_qualitygate_usergroup_association", &resource.Sweeper{
		Name: "sonarqube_qualitygate_usergroup_association",
		F:    testSweepSonarqubeQualitygateUsergroupAssociationSweeper,
	})
}

// TODO: implement sweeper to clean up projects: https://www.terraform.io/docs/extend/testing/acceptance-tests/sweepers.html
func testSweepSonarqubeQualitygateUsergroupAssociationSweeper(r string) error {
	return nil
}
func testAccPreCheckQualityGatePermissionFeature(t *testing.T) {
	sonarQubeVersion := testAccProvider.Meta().(*ProviderConfiguration).sonarQubeVersion

	minimumVersion, _ := version.NewVersion("9.2")
	if sonarQubeVersion.LessThan(minimumVersion) {
		t.Skipf("Skipping test of unsupported feature")
	}
}

func testAccSonarqubeQualitygateGroupAssociationGateName(rnd string, name string) string {
	return fmt.Sprintf(`
		resource "sonarqube_group" "%[1]s" {
			name = "%[2]s"
			description = "foo"
		}

		resource "sonarqube_qualitygate" "%[1]s" {
			name = "%[2]s"

			condition {
				metric    = "new_coverage"
				op        = "LT"
				threshold = "30"
			}
		}

		resource "sonarqube_qualitygate_usergroup_association" "%[1]s" {
			gatename   = sonarqube_qualitygate.%[1]s.name
			group_name = sonarqube_group.%[1]s.name
		}`, rnd, name)
}

func TestAccSonarqubeQualitygateGroupAssociationGateName(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_qualitygate_usergroup_association." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t); testAccPreCheckQualityGatePermissionFeature(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeQualitygateGroupAssociationGateName(rnd, "ping"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "gatename", "ping"),
					resource.TestCheckResourceAttr(name, "group_name", "ping"),
				),
			},
		},
	})
}

func testAccSonarqubeQualitygateUserAssociationGateName(rnd string, name string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "%[2]s"
			password   = "secret-Sauce37!"
		}

		resource "sonarqube_qualitygate" "%[1]s" {
			name = "%[2]s"

			condition {
				metric    = "new_coverage"
				op        = "LT"
				threshold = "30"
			}
		}

		resource "sonarqube_qualitygate_usergroup_association" "%[1]s" {
			gatename   = sonarqube_qualitygate.%[1]s.name
			login_name = sonarqube_user.%[1]s.name
		}`, rnd, name)
}

func TestAccSonarqubeQualitygateUserAssociationGateName(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_qualitygate_usergroup_association." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t); testAccPreCheckQualityGatePermissionFeature(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeQualitygateUserAssociationGateName(rnd, "pong"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "gatename", "pong"),
					resource.TestCheckResourceAttr(name, "login_name", "pong"),
				),
			},
		},
	})
}
