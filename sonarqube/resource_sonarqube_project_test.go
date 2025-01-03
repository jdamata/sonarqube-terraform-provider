package sonarqube

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func init() {
	resource.AddTestSweepers("sonarqube_project", &resource.Sweeper{
		Name: "sonarqube_project",
		F:    testSweepSonarqubeProjectSweeper,
	})
}

// TODO: implement sweeper to clean up projects: https://www.terraform.io/docs/extend/testing/acceptance-tests/sweepers.html
func testSweepSonarqubeProjectSweeper(r string) error {
	return nil
}

func testAccSonarqubeProjectBasicConfig(rnd string, name string, project string, visibility string) string {
	return fmt.Sprintf(`
		resource "sonarqube_project" "%[1]s" {
		  name       = "%[2]s"
		  project    = "%[3]s"
		  visibility = "%[4]s"
		}
		`, rnd, name, project, visibility)
}

func testAccSonarqubeProjectTagsConfig(rnd string, name string, project string, visibility string, tags []string) string {
	formattedTags := generateHCLList(tags)
	return fmt.Sprintf(`
		resource "sonarqube_project" "%[1]s" {
		  name       = "%[2]s"
		  project    = "%[3]s"
		  visibility = "%[4]s"
		  tags 		 = %[5]s // Note that the "" should be missing since this is a list
		}
		`, rnd, name, project, visibility, formattedTags)
}

func testAccSonarqubeProjectSettingsConfig(rnd string, name string, project string, visibility string, value string) string {
	return fmt.Sprintf(`
		resource "sonarqube_project" "%[1]s" {
		  name       = "%[2]s"
		  project    = "%[3]s"
		  visibility = "%[4]s"

		  setting {
			key   = "sonar.docker.activate"
			value = "%[5]s"
		  }
		}
		`, rnd, name, project, visibility, value)
}

func testAccSonarqubeProjectSettingsFieldValuesArray(rnd string, name string, project string, valuesKey string, valuesData []string) string {
	return fmt.Sprintf(`
	resource "sonarqube_project" "%[1]s" {
	  name       = "%[2]s"
	  project    = "%[3]s"
	  visibility = "public"

	  setting {
	    key   		 = "sonar.issue.ignore.multicriteria"
	    field_values = [
	      {
	        "resourceKey" = "src/main/java/**/*"
	        "ruleKey"     = "java:S1106"
	      },
	      {
	        "resourceKey" = "src/main/java/**/*"
	        "ruleKey"     = "java:S1120"
	      }
	    ]
	  }

	  setting {
	  	key    = "%[4]s"
		values = %[5]s
	  }

	}
	`, rnd, name, project, valuesKey, generateHCLList(valuesData))
}

func testAccSonarqubeProjectSettingsMultiple(rnd string, key string, name string, values []string, fields map[string]string) string {
	formattedValues := generateHCLList(values)
	formattedFields := generateHCLMap(fields)
	return fmt.Sprintf(`
	resource "sonarqube_project" "%[1]s" {
		name       = "%[2]s"
		project    = "%[3]s"
		visibility = "public"

		setting {
			key   = "sonar.terraform.activate"
			value = "true"
		}

		setting {
			key    = "sonar.terraform.file.suffixes"
			values = %[4]s
		}

		setting {
			key          = "sonar.issue.ignore.multicriteria"
			field_values = [%[5]s]
		}
	
	}
	`, rnd, key, name, formattedValues, formattedFields)
}

func TestAccSonarqubeProjectBasic(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "visibility", "public"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "visibility", "public"),
				),
			},
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "tf-postfix", "tf-postfix", "public"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "tf-postfix"),
					resource.TestCheckResourceAttr(name, "visibility", "public"),
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "tf-postfix"),
					resource.TestCheckResourceAttr(name, "visibility", "public"),
				),
			},
		},
	})
}

func TestAccSonarqubeProjectVisibilityUpdate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "visibility", "public"),
				),
			},
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "private"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "visibility", "private"),
				),
				// This is set to plan only as the admin user does not have the required privileges to make this change.
				// Requires 'Project administer' permission on the specified project or view
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccSonarqubeProjectTagsCreate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd
	tags := []string{"tag1", "tag2"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectTagsConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public", tags),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "tags.0", tags[0]),
					resource.TestCheckResourceAttr(name, "tags.1", tags[1]),
				),
			},
		},
	})
}

func TestAccSonarqubeProjectTagsUpdate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd
	tags := []string{"tag1", "tag2"}
	tagsUpdated := []string{"tag1", "tag2", "tag3"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "tags.#", "0"),
				),
			},
			{
				Config: testAccSonarqubeProjectTagsConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public", tags),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "tags.0", tags[0]),
					resource.TestCheckResourceAttr(name, "tags.1", tags[1]),
					resource.TestCheckResourceAttr(name, "tags.#", "2"),
				),
			},
			{
				Config: testAccSonarqubeProjectTagsConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public", tagsUpdated),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "tags.0", tagsUpdated[0]),
					resource.TestCheckResourceAttr(name, "tags.1", tagsUpdated[1]),
					resource.TestCheckResourceAttr(name, "tags.2", tagsUpdated[2]),
					resource.TestCheckResourceAttr(name, "tags.#", "3"),
				),
			},
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "tags.#", "0"),
				),
			},
		},
	})
}

func TestAccSonarqubeProjectKeyUpdate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd

	oldKey := "testAccSonarqubeProjectOld"
	newKey := "testAccSonarqubeProjectNew"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "testAccSonarqubeProject", oldKey, "public"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", oldKey),
				),
			},
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "testAccSonarqubeProject", newKey, "public"),

				// Make sure the update is in-place (https://developer.hashicorp.com/terraform/plugin/testing/acceptance-tests/plan-checks#examples-using-plancheck-expectresourceaction)
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(name, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", newKey),
				),
			},
		},
	})
}

func TestAccSonarqubeProjectSettingsCreate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd
	expectedSettings := 1

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectSettingsConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public", "false"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "setting.#", strconv.Itoa(expectedSettings)),
					resource.TestCheckResourceAttr(name, "setting.0.key", "sonar.docker.activate"),
					resource.TestCheckResourceAttr(name, "setting.0.value", "false"),
				),
			},
		},
	})
}

func TestAccSonarqubeProjectSettingsUpdate(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectBasicConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "setting.#", "0"),
				),
			},
			{
				Config: testAccSonarqubeProjectSettingsConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public", "false"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "setting.#", "1"),
					resource.TestCheckResourceAttr(name, "setting.0.key", "sonar.docker.activate"),
					resource.TestCheckResourceAttr(name, "setting.0.value", "false"),
				),
			},
			{
				Config: testAccSonarqubeProjectSettingsConfig(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", "public", "true"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "setting.#", "1"),
					resource.TestCheckResourceAttr(name, "setting.0.key", "sonar.docker.activate"),
					resource.TestCheckResourceAttr(name, "setting.0.value", "true"),
				),
			},
		},
	})
}

func TestAccSonarqubeProjectSettingsTypes(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd
	expectedConditions := 3
	values := []string{".tf", ".tfvars"}
	fieldValues := map[string]string{"ruleKey": "foo", "resourceKey": "bar"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectSettingsMultiple(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", values, fieldValues),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "setting.#", strconv.Itoa(expectedConditions)),
					resource.TestCheckResourceAttr(name, "setting.0.key", "sonar.terraform.activate"),
					resource.TestCheckResourceAttr(name, "setting.0.value", "true"),
					resource.TestCheckResourceAttr(name, "setting.1.key", "sonar.terraform.file.suffixes"),
					resource.TestCheckTypeSetElemAttr(name, "setting.1.values.*", ".tf"),
					resource.TestCheckTypeSetElemAttr(name, "setting.1.values.*", ".tfvars"),
					resource.TestCheckResourceAttr(name, "setting.2.key", "sonar.issue.ignore.multicriteria"),
					resource.TestCheckTypeSetElemNestedAttrs(name, "setting.2.field_values.*", fieldValues),
				),
			},
		},
	})
}

func TestAccSonarqubeProjectSettingsFieldValues(t *testing.T) {
	rnd := generateRandomResourceName()
	name := "sonarqube_project." + rnd

	var settingValuesKey string
	var settingValuesKeyData []string
	// Some settings are not available in community edition
	if strings.ToLower(testAccProvider.Meta().(*ProviderConfiguration).sonarQubeEdition) == "community" {
		settingValuesKey = "sonar.terraform.file.suffixes"
		settingValuesKeyData = []string{".tf", ".tfvars", ".hcl"}
	} else {
		settingValuesKey = "sonar.dbcleaner.branchesToKeepWhenInactive"
		settingValuesKeyData = []string{"master", "main", "release/*"}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccSonarqubeProjectSettingsFieldValuesArray(rnd, "testAccSonarqubeProject", "testAccSonarqubeProject", settingValuesKey, settingValuesKeyData),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "project", "testAccSonarqubeProject"),
					resource.TestCheckResourceAttr(name, "setting.#", strconv.Itoa(2)),
					resource.TestCheckResourceAttr(name, "setting.0.key", "sonar.issue.ignore.multicriteria"),
					resource.TestCheckResourceAttr(name, "setting.0.field_values.0.resourceKey", "src/main/java/**/*"),
					resource.TestCheckResourceAttr(name, "setting.0.field_values.1.resourceKey", "src/main/java/**/*"),
					resource.TestCheckResourceAttr(name, "setting.0.field_values.0.ruleKey", "java:S1106"),
					resource.TestCheckResourceAttr(name, "setting.0.field_values.1.ruleKey", "java:S1120"),
					resource.TestCheckResourceAttr(name, "setting.1.key", settingValuesKey),
					resource.TestCheckResourceAttr(name, "setting.1.values.#", strconv.Itoa(3)),
					resource.TestCheckResourceAttr(name, "setting.1.values.0", settingValuesKeyData[0]),
					resource.TestCheckResourceAttr(name, "setting.1.values.1", settingValuesKeyData[1]),
					resource.TestCheckResourceAttr(name, "setting.1.values.2", settingValuesKeyData[2]),
				),
			},
		},
	})

}
