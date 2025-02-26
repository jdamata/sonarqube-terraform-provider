package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubeGroupsDataSourceConfig(rnd string, name string, description string) string {
	return fmt.Sprintf(`
		resource "sonarqube_group" "%[1]s" {
			name        = "%[2]s"
			description = "%[3]s"
		}
		
		data "sonarqube_groups" "%[1]s" {
			search = sonarqube_group.%[1]s.name
		}`, rnd, name, description)
}

func TestAccSonarqubeGroupsDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_groups." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeGroupsDataSourceConfig(rnd, "testAccSonarqubeGroupsDataSource", "Terraform Test Groups Data-source"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "groups.#", "1"),
					resource.TestCheckResourceAttr(name, "groups.0.name", "testAccSonarqubeGroupsDataSource"),
					resource.TestCheckResourceAttr(name, "groups.0.description", "Terraform Test Groups Data-source"),
				),
			},
		},
	})
}
