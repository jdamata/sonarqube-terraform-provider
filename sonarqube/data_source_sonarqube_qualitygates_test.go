package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubeQualityGatesDataSourceConfig(rnd string, name string, metric string, op string, threshold string) string {
	return fmt.Sprintf(`
		resource "sonarqube_qualitygate" "%[1]s" {
			name      = "%[2]s"
			condition {
				metric    = "%[3]s"
				op        = "%[4]s"
				threshold = "%[5]s"
			}
		}
		
		data "sonarqube_qualitygates" "%[1]s" {
			name = sonarqube_qualitygate.%[1]s.name
		}`, rnd, name, metric, op, threshold)
}

func TestAccSonarqubeQualityGatesDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_qualitygates." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeQualityGatesDataSourceConfig(rnd, "testAccSonarqubeQualityGatesDataSource", "new_coverage", "LT", "50"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "quality_gates.#", "1"),
					resource.TestCheckResourceAttr(name, "quality_gates.0.name", "testAccSonarqubeQualityGatesDataSource"),
					resource.TestCheckResourceAttr(name, "quality_gates.0.condition.0.metric", "new_coverage"),
					resource.TestCheckResourceAttr(name, "quality_gates.0.condition.0.op", "LT"),
					resource.TestCheckResourceAttr(name, "quality_gates.0.condition.0.threshold", "50"),
				),
			},
		},
	})
}
