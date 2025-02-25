package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_permissions", &resource.Sweeper{
		Name: "sonarqube_permissions",
		F:    testSweepPermissionSweeper,
	})
}

// TODO: implement sweeper to clean up permission_template: https://www.terraform.io/docs/extend/testing/acceptance-tests/sweepers.html
func testSweepPermissionSweeper(r string) error {
	return nil
}

func testAccSonarqubePermissionGroupNameConfig(id string, groupName string, permissions []string) string {
	formattedPermissions := generateHCLList(permissions)
	return fmt.Sprintf(`
		resource "sonarqube_group" "%[1]s" {
			name        = "%[2]s"
			description = "%[2]s"
		}

		resource "sonarqube_permissions" "%[1]s" {
			group_name  = sonarqube_group.%[1]s.name
			permissions = %[3]s
		}`, id, groupName, formattedPermissions)
}

func TestAccSonarqubePermissionGroupName(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_permissions." + rnd
	permissions := []string{"admin"}
	updatedPermissions := []string{"admin", "profileadmin"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubePermissionGroupNameConfig(rnd, "testAccSonarqubePermissions", permissions),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "group_name", "testAccSonarqubePermissions"),
				),
			},
			{
				Config: testAccSonarqubePermissionGroupNameConfig(rnd, "testAccSonarqubePermissions", updatedPermissions),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "group_name", "testAccSonarqubePermissions"),
				),
			},
		},
	})
}

func testAccSonarqubePermissionLoginNameConfig(id string, loginName string, permissions []string) string {
	formattedPermissions := generateHCLList(permissions)
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "%[2]s"
			password   = "secret-Sauce37!"
		}

		resource "sonarqube_permissions" "%[1]s" {
		  	login_name  = sonarqube_user.%[1]s.name
		  	permissions = %[3]s
		}
		`, id, loginName, formattedPermissions)
}

func TestAccSonarqubePermissionLoginName(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_permissions." + rnd
	permissions := []string{"gateadmin", "profileadmin"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubePermissionLoginNameConfig(rnd, "testAccSonarqubePermissions", permissions),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "login_name", "testAccSonarqubePermissions"),
					//resource.TestCheckResourceAttr(name, "permissions", generateHCLList(permissions)),
				),
			},
		},
	})
}

func testAccSonarqubePermissionLoginNameTemplateNameConfig(id string, loginName string, permissions []string) string {
	formattedPermissions := generateHCLList(permissions)
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "%[2]s"
			password   = "secret-Sauce37!"
		}

		resource "sonarqube_permission_template" "this" {
			name = "foo"
		}
		resource "sonarqube_permissions" "%[1]s" {
		  	login_name    = sonarqube_user.%[1]s.name
			template_name = sonarqube_permission_template.this.name
		  	permissions   = %[3]s
		}
		`, id, loginName, formattedPermissions)
}

func TestAccSonarqubePermissionLoginNameTemplateName(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_permissions." + rnd
	permissions := []string{"codeviewer", "scan"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubePermissionLoginNameTemplateNameConfig(rnd, "testAccSonarqubePermissions", permissions),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "login_name", "testAccSonarqubePermissions"),
				),
			},
		},
	})
}
