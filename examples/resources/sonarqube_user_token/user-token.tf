resource "sonarqube_user" "user" {
  login_name = "terraform-test"
  name       = "terraform-test"
  password   = "secret-Sauce37!"
}

resource "sonarqube_user_token" "token" {
  login_name = sonarqube_user.user.login_name
  name       = "my-token"
}

output "user_token" {
  value = sonarqube_user_token.token.token
}
