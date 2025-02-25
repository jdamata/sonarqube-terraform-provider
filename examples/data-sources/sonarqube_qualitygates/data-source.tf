data "sonarqube_qualitygates" "qualitygates" {

}

data "sonarqube_qualitygates" "qualitygates_sonarway" {
  name           = "Sonar way"
  ignore_missing = true
}
