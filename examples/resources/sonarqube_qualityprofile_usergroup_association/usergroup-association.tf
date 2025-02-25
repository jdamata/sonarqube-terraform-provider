resource "sonarqube_qualityprofile" "main" {
  name     = "my_qualityprofile"
  language = "java"
  parent   = "Sonar way"
}

resource "sonarqube_group" "qa_team" {
  name        = "QA-Team"
  description = "Quality Assurence Team"
}

resource "sonarqube_qualityprofile_usergroup_association" "main" {
  profile_name = sonarqube_qualityprofile.main.id
  language     = sonarqube_qualityprofile.language.id
  group_name   = sonarqube_group.qa_team.name
}
