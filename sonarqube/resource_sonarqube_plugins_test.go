package sonarqube

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_plugin", &resource.Sweeper{
		Name: "sonarqube_plugin",
		F:    testSweepSonarqubePluginSweeper,
	})
}

// TODO: implement sweeper to clean up projects: https://www.terraform.io/docs/extend/testing/acceptance-tests/sweepers.html
func testSweepSonarqubePluginSweeper(r string) error {
	return nil
}

func testAccSonarqubePluginBasicConfig(rnd string, key string) string {
	return fmt.Sprintf(`
		resource "sonarqube_plugin" "%[1]s" {
				key = "%[2]s"
		}`, rnd, key)
}

func TestAccSonarqubePluginBasic(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_plugin." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubePluginBasicConfig(rnd, "cloudformation"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "key", "cloudformation"),
				),
				// Because sonarqube must be restarted to finish installing a plugin, we should expect an error here when trying to delete the plugin
				// Won'thttps://github.com/hashicorp/terraform-plugin-sdk/issues/609
				ExpectError: regexp.MustCompile("resourceSonarqubePluginDelete: Failed to delete plugin: API returned an error: Plugin [cloudformation] is not installed"),
			},
		},
	})
}
