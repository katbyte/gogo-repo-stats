package gh

import "strings"

func RepoShortName(repo string) string {
	repoShort := strings.Split(repo, "/")[1] // remove owner by default
	switch repo {
	case "hashicorp/terraform-provider-azurerm":
		repoShort = "azurerm"
	case "hashicorp/terraform-provider-azuread":
		repoShort = "azuread"
	case "hashicorp/terraform-provider-azurestack":
		repoShort = "azurestack"
	}
	return repoShort
}
