data "sonarqube_qualitygates" "qualitygates" {

}

data "sonarqube_qualitygates" "qualitygates_sonarway" {
  search         = "Sonar way"
  ignore_missing = true
}
