package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccSonarqubeLanguagesDataSourceConfig(rnd string, languageKey string) string {
	return fmt.Sprintf(`
		data "sonarqube_languages" "%[1]s" {
			search = "%[2]s"
		}`, rnd, languageKey)
}

func TestAccSonarqubeLanguagesDataSource(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "data.sonarqube_languages." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeLanguagesDataSourceConfig(rnd, "terraform"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "languages.#", "1"),
					resource.TestCheckResourceAttr(name, "languages.0.key", "terraform"),
					resource.TestCheckResourceAttr(name, "languages.0.name", "Terraform"),
				),
			},
		},
	})
}
