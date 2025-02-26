package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubePermissionTemplatesDataSourceConfig(rnd string, name string, description string, projectKeyPattern string) string {
	return fmt.Sprintf(`
		resource "sonarqube_permission_template" "%[1]s" {
		  name                = "%[2]s"
		  description         = "%[3]s"
		  project_key_pattern = "%[4]s"
		}
		
		data "sonarqube_permission_templates" "%[1]s" {
			search = sonarqube_permission_template.%[1]s.name
		}`, rnd, name, description, projectKeyPattern)
}

func TestAccSonarqubePermissionTemplatesDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_permission_templates." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubePermissionTemplatesDataSourceConfig(rnd, "testAccSonarqubePermissionTemplatesDataSource", "These are internal projects", "internal.*"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "permission_templates.#", "1"),
					resource.TestCheckResourceAttr(name, "permission_templates.0.name", "testAccSonarqubePermissionTemplatesDataSource"),
					resource.TestCheckResourceAttr(name, "permission_templates.0.description", "These are internal projects"),
					resource.TestCheckResourceAttr(name, "permission_templates.0.project_key_pattern", "internal.*"),
				),
			},
		},
	})
}
