package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_rule", &resource.Sweeper{
		Name: "sonarqube_rule",
		F:    testSweepSonarqubeRuleSweeper,
	})
}

func testSweepSonarqubeRuleSweeper(r string) error {
	return nil
}

func testAccSonarqubeRuleBasicConfig(rnd string, key string, markdown_description string, name string, template_key string) string {
	return fmt.Sprintf(`
		resource "sonarqube_rule" "%[1]s" {
			custom_key = "%[2]s"
			markdown_description = "%[3]s"
			name = "%[4]s"
			template_key = "%[5]s"
		}`, rnd, key, markdown_description, name, template_key)
}

func TestAccSonarqubeRuleBasic(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_rule." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeRuleBasicConfig(rnd, "key", "markdown_description", "name", "xml:XPathCheck"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "custom_key", "key"),
					resource.TestCheckResourceAttr(name, "key", "key"),
					resource.TestCheckResourceAttr(name, "markdown_description", "markdown_description"),
					resource.TestCheckResourceAttr(name, "name", "name"),
					resource.TestCheckResourceAttr(name, "template_key", "xml:XPathCheck"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "custom_key", "key"),
					resource.TestCheckResourceAttr(name, "key", "key"),
					resource.TestCheckResourceAttr(name, "markdown_description", "markdown_description"),
					resource.TestCheckResourceAttr(name, "params", "key1=v1"),
					resource.TestCheckResourceAttr(name, "prevent_reactivation", "true"),
					resource.TestCheckResourceAttr(name, "severity", "INFO"),
					resource.TestCheckResourceAttr(name, "status", "BETA"),
					resource.TestCheckResourceAttr(name, "template_key", "xml:XPathCheck"),
				),
			},
		},
	})
}
