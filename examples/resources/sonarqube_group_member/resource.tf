resource "sonarqube_user" "user" {
  login_name = "terraform-test"
  name       = "terraform-test"
  password   = "secret-Sauce37!"
}

resource "sonarqube_group" "project_users" {
  name        = "Project-Users"
  description = "This is a group"
}

resource "sonarqube_group_member" "project_users_member" {
  name       = sonarqube_group.project_users.name
  login_name = sonarqube_user.user.login_name
}
