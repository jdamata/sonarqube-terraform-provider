data "sonarqube_qualityprofiles" "qualityprofiles" {

}

data "sonarqube_qualityprofiles" "qualityprofiles_sonarway_java" {
  name     = "Sonar way"
  language = "java"
}
