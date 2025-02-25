package sonarqube

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func init() {
	resource.AddTestSweepers("sonarqube_user", &resource.Sweeper{
		Name: "sonarqube_user",
		F:    testSweepSonarqubeUserSweeper,
	})
}

func testSweepSonarqubeUserSweeper(r string) error {
	return nil
}

func testAccSonarqubeUserLocalConfig(rnd string, name string, email string, password string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "%[2]s"
			email      = "%[3]s"
			password   = "%[4]s"
		}`, rnd, name, email, password)
}

func TestAccSonarqubeUserLocal(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_user." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserLocalConfig(rnd, "testAccSonarqubeUserLocal", "terraform-test@sonarqube.com", "secret-Sauce37!"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserLocal"),
					resource.TestCheckResourceAttr(name, "email", "terraform-test@sonarqube.com"),
				),
			},
			{
				Config: testAccSonarqubeUserLocalConfig(rnd, "testAccSonarqubeUserLocal", "terraform-test2@sonarqube.com", "secret-Sauce38!"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserLocal"),
					resource.TestCheckResourceAttr(name, "email", "terraform-test2@sonarqube.com"),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserLocal"),
				),
			},
		},
	})
}

func testAccSonarqubeUserNotLocalConfig(rnd string, name string, email string) string {
	return fmt.Sprintf(`
		resource "sonarqube_user" "%[1]s" {
			login_name = "%[2]s"
			name       = "%[2]s"
			email      = "%[3]s"
			is_local   = false
	    }`, rnd, name, email)
}

func TestAccSonarqubeUserNotLocal(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_user." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeUserNotLocalConfig(rnd, "testAccSonarqubeUserNotLocal", "terraform-test@sonarqube.com"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserNotLocal"),
					resource.TestCheckResourceAttr(name, "email", "terraform-test@sonarqube.com"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "testAccSonarqubeUserNotLocal"),
				),
			},
		},
	})
}
