package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-version"
	"github.com/manifoldco/promptui"
	log "github.com/sirupsen/logrus"
)

type SubLabel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func GetAzCliVersion() string {
	azCmd := exec.Command("az", "version", "-o", "json")
	out, err := azCmd.CombinedOutput()
	if err != nil {
		log.Fatal("Error: unable to obtain az cli version")
	}

	var version map[string]interface{}
	if err := json.Unmarshal(out, &version); err != nil {
		log.Fatal("unable to unmarshal az cli version output to map")
	}

	return fmt.Sprint(version["azure-cli"])
}

func getAzUpgrade() string {
	selection := &promptui.Select{
		Label: "Your Azure CLI version must be at least 2.37.0 - would you like us to update it for you?",
		Items: []string{"yes", "no"},
	}

	_, selectResponse, err := selection.Run()
	if err != nil {
		return err.Error()
	}

	return selectResponse
}

func upgradeAzCli() {
	azCmd := exec.Command("az", "upgrade", "-y")
	_, err := azCmd.CombinedOutput()
	if err != nil {
		log.Fatal("Error: unable to upgrade az cli version; ", err)
	}

	log.Info("Azure CLI upgrade was successful!")
}

func CheckAzCliInstalled() {
	log.Debug("Checking that Azure Cli is installed...")
	azCmd := exec.Command("az")
	_, err := azCmd.CombinedOutput()
	if err != nil {
		log.Fatal("Error: AZ cli not installed. Find installation instructions at this link: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli")
	}

	currentVersion, err := version.NewVersion(GetAzCliVersion())
	if err != nil {
		log.Fatal(err)
	}

	constraints, err := version.NewConstraint(">= 2.37")
	if err != nil {
		log.Fatal(err)
	}

	if !constraints.Check(currentVersion) {
		if ans := getAzUpgrade(); ans == "no" {
			log.Fatal("Az cli version must be at least 2.37.0")
		}
		upgradeAzCli()
	}
}

func IsLoggedInToAz() bool {
	log.Debug("Checking that user is logged in to Azure CLI...")
	azCmd := exec.Command("az", "ad", "signed-in-user", "show", "--only-show-errors", "--query", "objectId")
	_, err := azCmd.CombinedOutput()
	if err != nil {
		return false
	}

	return true
}

func HasGhCli() bool {
	log.Debug("Checking that github cli is installed...")
	ghCmd := exec.Command("gh")
	_, err := ghCmd.CombinedOutput()
	if err != nil {
		log.Fatal("Error: The github cli is required to complete this process. Find installation instructions at this link: https://github.com/cli/cli#installation")
		return false
	}

	log.Debug("Github cli found!")
	return true
}

func IsLoggedInToGh() bool {
	log.Debug("Checking that user is logged in to github...")
	ghCmd := exec.Command("gh", "auth", "status")
	out, err := ghCmd.CombinedOutput()
	if err != nil {
		fmt.Printf(string(out))
		return false
	}

	log.Debug("User is logged in!")
	return true

}

func LogInToGh() error {
	log.Debug("Logging user in to github...")
	ghCmd := exec.Command("gh", "auth", "login")
	ghCmd.Stdin = os.Stdin
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr
	err := ghCmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func LogInToAz() error {
	log.Debug("Logging user in to Azure Cli...")
	azCmd := exec.Command("az", "login", "--allow-no-subscriptions")
	azCmd.Stdin = os.Stdin
	azCmd.Stdout = os.Stdout
	azCmd.Stderr = os.Stderr
	err := azCmd.Run()
	if err != nil {
		return err
	}

	log.Debug("Successfully logged in!")
	return nil
}

func IsSubscriptionIdValid(subscriptionId string) error {
	if subscriptionId == "" {
		return errors.New("subscriptionId cannot be empty")
	}

	getSubscriptionIdCmd := exec.Command("az", "account", "show", "-s", subscriptionId, "--query", "id")
	out, err := getSubscriptionIdCmd.CombinedOutput()
	if err != nil {
		return err
	}

	var azSubscription string
	if err = json.Unmarshal(out, &azSubscription); err != nil {
		return err
	}

	if azSubscription == "" {
		return errors.New("subscription not found")
	}

	return nil
}

func isValidResourceGroup(
	subscriptionId string,
	resourceGroup string,
) error {
	if resourceGroup == "" {
		return errors.New("resource group cannot be empty")
	}

	query := fmt.Sprintf("[?name=='%s']", resourceGroup)
	getResourceGroupCmd := exec.Command("az", "group", "list", "--subscription", subscriptionId, "--query", query)
	out, err := getResourceGroupCmd.CombinedOutput()
	if err != nil {
		log.Errorf("failed to validate resource group %q from subscription %q: %s", resourceGroup, subscriptionId, err)
		return err
	}

	var rg []interface{}
	if err = json.Unmarshal(out, &rg); err != nil {
		return err
	}

	if len(rg) == 0 {
		return fmt.Errorf("resource group %q not found from subscription %q", resourceGroup, subscriptionId)
	}

	return nil
}

func isValidGhRepo(repo string) error {
	listReposCmd := exec.Command("gh", "repo", "view", repo)
	_, err := listReposCmd.CombinedOutput()
	if err != nil {
		log.Fatal("Github repo not found")
		return err
	}
	return nil
}

func AzAppExists(appName string) bool {
	filter := fmt.Sprintf("displayName eq '%s'", appName)
	checkAppExistsCmd := exec.Command("az", "ad", "app", "list", "--only-show-errors", "--filter", filter, "--query", "[].appId")
	out, err := checkAppExistsCmd.CombinedOutput()
	if err != nil {
		return false
	}

	var azApp []string
	json.Unmarshal(out, &azApp)

	return len(azApp) >= 1
}

func (sc *SetUpCmd) ServicePrincipalExists() bool {
	checkSpExistsCmd := exec.Command("az", "ad", "sp", "show", "--only-show-errors", "--id", sc.appId, "--query", "id")
	out, err := checkSpExistsCmd.CombinedOutput()
	if err != nil {
		return false
	}

	var objectId string
	json.Unmarshal(out, &objectId)

	log.Debug("Service principal exists")
	// TODO: tell user sp already exists and ask if they want to use it?
	sc.spObjectId = objectId
	return true
}

func AzAcrExists(acrName string) bool {
	query := fmt.Sprintf("[?name=='%s']", acrName)
	checkAcrExistsCmd := exec.Command("az", "acr", "list", "--only-show-errors", "--query", query)
	out, err := checkAcrExistsCmd.CombinedOutput()
	if err != nil {
		return false
	}

	var azAcr []interface{}
	json.Unmarshal(out, &azAcr)

	if len(azAcr) >= 1 {
		return true
	}

	return false
}

func AzAksExists(aksName string, resourceGroup string) bool {
	checkAksExistsCmd := exec.Command("az", "aks", "browse", "-g", resourceGroup, "--name", aksName)
	_, err := checkAksExistsCmd.CombinedOutput()
	if err != nil {
		return false
	}

	return true
}

func GetCurrentAzSubscriptionLabel() (SubLabel, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return SubLabel{}, fmt.Errorf("failed to log in to Azure CLI: %v", err)
		}
	}

	getAccountCmd := exec.Command("az", "account", "show", "--query", "{id: id, name: name}")
	out, err := getAccountCmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}

	var currentSub SubLabel
	if err := json.Unmarshal(out, &currentSub); err != nil {
		return SubLabel{}, fmt.Errorf("failed to unmarshal JSON output: %v", err)
	} else if currentSub.ID == "" {
		return SubLabel{}, errors.New("no current subscription found")
	}

	return currentSub, nil
}

func GetAzSubscriptionLabels() ([]SubLabel, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %v", err)
		}
	}

	getAccountCmd := exec.Command("az", "account", "list", "--all", "--query", "[].{id: id, name: name}")

	out, err := getAccountCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("get azure subscription labels: %w", err)
	}

	var subLabels []SubLabel
	if err := json.Unmarshal(out, &subLabels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON output: %v", err)
	} else if len(subLabels) == 0 {
		return nil, errors.New("no subscriptions found")
	}

	return subLabels, nil
}

func GetAzResourceGroups() ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %v", err)
		}
	}

	getResourceGroupsCmd := exec.Command("az", "group", "list", "--query", "[].name")
	out, err := getResourceGroupsCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("get azure resource groups: %w", err)
	}

	var resourceGroups []string
	if err := json.Unmarshal(out, &resourceGroups); err != nil {
		return nil, fmt.Errorf("get azure resource groups: %v", err)
	} else if len(resourceGroups) == 0 {
		return nil, errors.New("no resource groups found")
	}

	return resourceGroups, nil
}

func GetAzContainerRegistries(resourceGroup string) ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %v", err)
		}
	}

	getAcrCmd := exec.Command("az", "acr", "list", "-g", resourceGroup, "--query", "[].name")
	out, err := getAcrCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("get azure container registries: %w", err)
	}

	var acrs []string
	if err := json.Unmarshal(out, &acrs); err != nil {
		return nil, fmt.Errorf("get azure container registries: %v", err)
	} else if len(acrs) == 0 {
		return nil, errors.New("no container registries found")
	}

	return acrs, nil
}

func GetAzContainerNames(containerRegistry string) ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %v", err)
		}
	}

	getContainerNameCmd := exec.Command("az", "acr", "repository", "list", "-n", containerRegistry)
	out, err := getContainerNameCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("get azure container name: %w", err)
	}

	var containers []string
	if err := json.Unmarshal(out, &containers); err != nil {
		return nil, fmt.Errorf("get azure container name: %v", err)
	} else if len(containers) == 0 {
		return nil, errors.New("no containers found")
	}

	return containers, nil
}

func GetAzClusters(resourceGroup string) ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %v", err)
		}
	}

	getClustersCmd := exec.Command("az", "aks", "list", "-g", resourceGroup, "--query", "[].name")
	out, err := getClustersCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("get azure clusters: %w", err)
	}

	var clusters []string
	if err := json.Unmarshal(out, &clusters); err != nil {
		return nil, fmt.Errorf("get azure clusters: %v", err)
	} else if len(clusters) == 0 {
		return nil, errors.New("no clusters found")
	}

	return clusters, nil
}
