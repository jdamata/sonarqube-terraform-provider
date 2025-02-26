package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubeQualityProfilesDataSourceConfig(rnd string, name string, language string) string {
	return fmt.Sprintf(`
		resource "sonarqube_qualityprofile" "%[1]s" {
			name     = "%[2]s"
			language = "%[3]s"
		}
		
		data "sonarqube_qualityprofiles" "%[1]s" {
			name = "%[2]s"
			language = "%[3]s"
			depends_on = [sonarqube_qualityprofile.%[1]s]
		}`, rnd, name, language)
}

func TestAccSonarqubeQualityProfilesDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_qualityprofiles." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeQualityProfilesDataSourceConfig(rnd, "testAccSonarqubeQualityProfilesDataSource", "js"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "quality_profiles.#", "1"),
					resource.TestCheckResourceAttr(name, "quality_profiles.0.name", "testAccSonarqubeQualityProfilesDataSource"),
					resource.TestCheckResourceAttr(name, "quality_profiles.0.language", "js"),
				),
			},
		},
	})
}
