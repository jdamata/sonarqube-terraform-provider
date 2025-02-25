resource "sonarqube_qualitygate" "main" {
  name = "my_qualitygate"

  condition {
    metric    = "new_coverage"
    op        = "LT"
    threshold = "30"
  }
}

resource "sonarqube_user" "qa_user" {
  login_name = "qa-user"
  name       = "qa-user"
  password   = "secret-Sauce37!"
}

resource "sonarqube_qualitygate_usergroup_association" "main" {
  gatename   = sonarqube_qualitygate.main.id
  login_name = sonarqube_user.qa_user.id
}
