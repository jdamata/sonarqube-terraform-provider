//go:build tools

package tools

import (
	// Documentation generation
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)

// * Run the docs generation tool
//go:generate echo "Generate provider documentation..."
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
