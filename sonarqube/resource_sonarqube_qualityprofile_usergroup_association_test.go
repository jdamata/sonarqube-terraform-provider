package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_qualityprofile_usergroup_association", &resource.Sweeper{
		Name: "sonarqube_qualityprofile_usergroup_association",
		F:    testSweepSonarqubeQualityprofileUsergroupAssociationSweeper,
	})
}

// TODO: implement sweeper to clean up projects: https://www.terraform.io/docs/extend/testing/acceptance-tests/sweepers.html
func testSweepSonarqubeQualityprofileUsergroupAssociationSweeper(r string) error {
	return nil
}
func testAccPreCheckQualityProfilePermissionFeature(t *testing.T) {
	sonarQubeVersion := testAccProvider.Meta().(*ProviderConfiguration).sonarQubeVersion

	minimumVersion, _ := version.NewVersion("9.2")
	if sonarQubeVersion.LessThan(minimumVersion) {
		t.Skipf("Skipping test of unsupported feature")
	}
}

func testAccSonarqubeQualityprofileGroupAssociationProfileName(rnd string, name string, language string) string {
	return fmt.Sprintf(`
		resource "sonarqube_group" "%[1]s" {
			name = "%[2]s"
			description = "foo"
		}

		resource "sonarqube_qualityprofile" "%[1]s" {
			name = "%[2]s"
			language = "%[3]s"
		}

		resource "sonarqube_qualityprofile_usergroup_association" "%[1]s" {
			profile_name   = sonarqube_qualityprofile.%[1]s.name
			language   = sonarqube_qualityprofile.%[1]s.language
			group_name = sonarqube_group.%[1]s.name
		}`, rnd, name, language)
}

func TestAccSonarqubeQualityprofileGroupAssociationProfileName(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_qualityprofile_usergroup_association." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t); testAccPreCheckQualityProfilePermissionFeature(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeQualityprofileGroupAssociationProfileName(rnd, "ping", "terraform"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "profile_name", "ping"),
					resource.TestCheckResourceAttr(name, "group_name", "ping"),
				),
			},
		},
	})
}

func testAccSonarqubeQualityprofileUserAssociationProfileName(rnd string, name string, language string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "%[2]s"
			password   = "secret-Sauce37!"
		}

		resource "sonarqube_qualityprofile" "%[1]s" {
			name = "%[2]s"
			language = "%[3]s"
		}

		resource "sonarqube_qualityprofile_usergroup_association" "%[1]s" {
			profile_name   = sonarqube_qualityprofile.%[1]s.name
			language   = sonarqube_qualityprofile.%[1]s.language
			login_name = sonarqube_user.%[1]s.name
		}`, rnd, name, language)
}

func TestAccSonarqubeQualityprofileUserAssociationProfileName(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_qualityprofile_usergroup_association." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t); testAccPreCheckQualityProfilePermissionFeature(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeQualityprofileUserAssociationProfileName(rnd, "pong", "terraform"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "profile_name", "pong"),
					resource.TestCheckResourceAttr(name, "login_name", "pong"),
				),
			},
		},
	})
}
