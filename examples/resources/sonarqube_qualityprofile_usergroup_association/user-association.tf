resource "sonarqube_qualityprofile" "main" {
  name     = "my_qualityprofile"
  language = "java"
  parent   = "Sonar way"
}

resource "sonarqube_user" "qa_user" {
  login_name = "qa-user"
  name       = "qa-user"
  password   = "secret-sauce37!"
}

resource "sonarqube_qualityprofile_usergroup_association" "main" {
  profile_name = sonarqube_qualityprofile.main.id
  language     = sonarqube_qualityprofile.language.id
  login_name   = sonarqube_user.qa_user.id
}
