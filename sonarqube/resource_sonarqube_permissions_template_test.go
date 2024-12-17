package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_permission_template", &resource.Sweeper{
		Name: "sonarqube_permission_template",
		F:    testSweepPermissionTemplateSweeper,
	})
}

// TODO: implement sweeper to clean up permission_template: https://www.terraform.io/docs/extend/testing/acceptance-tests/sweepers.html
func testSweepPermissionTemplateSweeper(r string) error {
	return nil
}

func testAccSonarqubePermissionTemplateBasicConfig(rnd string, name string, description string, projectKeyPattern string) string {
	return fmt.Sprintf(`
		resource "sonarqube_permission_template" "%[1]s" {
		  name                = "%[2]s"
		  description         = "%[3]s"
		  project_key_pattern = "%[4]s"
		}`, rnd, name, description, projectKeyPattern)
}

func testAccSonarqubePermissionTemplateDefaultTemplate(rnd string, name string, description string, projectKeyPattern string) string {
	return fmt.Sprintf(`
		resource "sonarqube_permission_template" "%[1]s" {
		  name                = "%[2]s"
		  description         = "%[3]s"
		  project_key_pattern = "%[4]s"
		  default             = true
		}`, rnd, name, description, projectKeyPattern)
}

func testAccSonarqubePermissionTemplateBulkApply(rnd string, name string, description string, projectKeyPattern string, bulkApply bool) string {
	return fmt.Sprintf(`
		resource "sonarqube_permission_template" "%[1]s" {
		  name                = "%[2]s"
		  description         = "%[3]s"
		  project_key_pattern = "%[4]s"
		  bulk_apply          = %[5]t
		}`, rnd, name, description, projectKeyPattern, bulkApply)
}

func TestAccSonarqubePermissionTemplateBasic(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_permission_template." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubePermissionTemplateBasicConfig(rnd, "testAccSonarqubePermissionTemplate", "These are internal projects", "internal.*"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubePermissionTemplate"),
					resource.TestCheckResourceAttr(name, "description", "These are internal projects"),
					resource.TestCheckResourceAttr(name, "project_key_pattern", "internal.*"),
				),
			},
			{
				Config: testAccSonarqubePermissionTemplateBasicConfig(rnd, "testAccSonarqubePermissionTemplate", "These are internal projects 2", "internal.*"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubePermissionTemplate"),
					resource.TestCheckResourceAttr(name, "description", "These are internal projects 2"),
					resource.TestCheckResourceAttr(name, "project_key_pattern", "internal.*"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubePermissionTemplate"),
					resource.TestCheckResourceAttr(name, "description", "These are internal projects"),
					resource.TestCheckResourceAttr(name, "project_key_pattern", "internal.*"),
				),
			},
		},
	})
}

func TestAccSonarqubePermissionTemplateDefaultTemplate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_permission_template." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubePermissionTemplateDefaultTemplate(rnd, "testAccSonarqubePermissionTemplateDefault", "These are internal projects", "internal.*"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubePermissionTemplateDefault"),
					resource.TestCheckResourceAttr(name, "description", "These are internal projects"),
					resource.TestCheckResourceAttr(name, "project_key_pattern", "internal.*"),
				),
				// Must be set to plan as its not possible to destroy a template that is the current default.
				// This results in the error: It is not possible to delete the default permission template for projects
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccSonarqubePermissionTemplateUpdate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_permission_template." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubePermissionTemplateBulkApply(rnd, "testAccSonarqubePermissionTemplateUpdate", "Initial description", "initial.*", false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubePermissionTemplateUpdate"),
					resource.TestCheckResourceAttr(name, "description", "Initial description"),
					resource.TestCheckResourceAttr(name, "project_key_pattern", "initial.*"),
				),
			},
			{
				Config: testAccSonarqubePermissionTemplateBulkApply(rnd, "testAccSonarqubePermissionTemplateUpdate", "Updated description", "updated.*", true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubePermissionTemplateUpdate"),
					resource.TestCheckResourceAttr(name, "description", "Updated description"),
					resource.TestCheckResourceAttr(name, "project_key_pattern", "updated.*"),
					resource.TestCheckResourceAttr(name, "bulk_apply", "true"),
				),
			},
		},
	})
}
