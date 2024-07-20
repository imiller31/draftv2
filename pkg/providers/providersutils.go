package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"time"

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
	loading := make(chan bool)
	go showLoader(loading)

	out, err := azCmd.CombinedOutput()
	loading <- true
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
	loading := make(chan bool)
	go showLoader(loading)

	_, err := azCmd.CombinedOutput()
	loading <- true
	if err != nil {
		log.Fatal("Error: unable to upgrade az cli version; ", err)
	}

	log.Info("Azure CLI upgrade was successful!")
}

func CheckAzCliInstalled() {
	log.Debug("Checking that Azure Cli is installed...")
	azCmd := exec.Command("az")
	loading := make(chan bool)
	go showLoader(loading)

	_, err := azCmd.CombinedOutput()
	loading <- true
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

func checkKubectlVersion(clusterResourceGroup, clusterName string) error {
	getKubectlVersionCmd := exec.Command("az", "aks", "command", "invoke", "-g", clusterResourceGroup, "-n", clusterName, "--command", "kubectl version -o json", "-o", "json")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getKubectlVersionCmd.CombinedOutput()
	loading <- true

	if err != nil {
		return fmt.Errorf("failed to obtain kubectl version: %w", err)
	}

	var jsonData struct {
		LogData string `json:"logs"`
	}

	if err := json.Unmarshal(out, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal kubectl version: %w", err)
	}

	type Version struct {
		Major string `json:"major"`
		Minor string `json:"minor"`
	}

	var kubectlVersion struct {
		Client Version `json:"clientVersion"`
		Server Version `json:"serverVersion"`
	}

	if err := json.Unmarshal([]byte(jsonData.LogData), &kubectlVersion); err != nil {
		return fmt.Errorf("failed to unmarshal logs data: %w", err)
	}

	clientMajorVersion, err := strconv.ParseFloat(kubectlVersion.Client.Major, 64)
	if err != nil {
		return fmt.Errorf("failed to parse client major version: %w", err)
	}

	clientMinorVersion, err := strconv.ParseFloat(kubectlVersion.Client.Minor, 64)
	if err != nil {
		return fmt.Errorf("failed to parse client minor version: %w", err)
	}

	clientVersion := clientMajorVersion + clientMinorVersion/100

	serverMajorVersion, err := strconv.ParseFloat(kubectlVersion.Server.Major, 64)
	if err != nil {
		return fmt.Errorf("failed to parse server major version: %w", err)
	}

	serverMinorVersion, err := strconv.ParseFloat(kubectlVersion.Server.Minor, 64)
	if err != nil {
		return fmt.Errorf("failed to parse server minor version: %w", err)
	}

	serverVersion := serverMajorVersion + serverMinorVersion/100

	versionDiff := math.Abs(clientVersion - serverVersion)
	if versionDiff > 0.1 {
		log.Infof("Warning: kubectl client version %v differs from kubectl server version %v by more than 1 minor version, this may lead to incompatibility issues\n", clientVersion, serverVersion)
	}

	return nil
}

func checkKubectlInstalled(clusterResourceGroup, clusterName string) error {
	log.Debug("Checking that kubectl is installed...")
	kubectlCmd := exec.Command("kubectl")
	loading1 := make(chan bool)
	go showLoader(loading1)

	_, err := kubectlCmd.CombinedOutput()
	loading1 <- true

	if err != nil {
		return errors.New("kubectl not installed:\nFind installation instructions at this link: https://kubernetes.io/docs/tasks/tools/install-kubectl/")
	}

	return checkKubectlVersion(clusterResourceGroup, clusterName)
}

func IsLoggedInToAz() bool {
	log.Debug("Checking that user is logged in to Azure CLI...")
	azCmd := exec.Command("az", "ad", "signed-in-user", "show", "--only-show-errors", "--query", "objectId")
	loading := make(chan bool)
	go showLoader(loading)
	_, err := azCmd.CombinedOutput()
	loading <- true

	return err == nil
}

func HasGhCli() bool {
	log.Debug("Checking that github cli is installed...")
	ghCmd := exec.Command("gh")
	loading := make(chan bool)
	go showLoader(loading)

	_, err := ghCmd.CombinedOutput()
	loading <- true
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
	loading := make(chan bool)
	go showLoader(loading)

	out, err := ghCmd.CombinedOutput()
	loading <- true
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
	loading := make(chan bool)
	go showLoader(loading)
	ghCmd.Stdin = os.Stdin
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr
	err := ghCmd.Run()
	loading <- true
	if err != nil {
		return err
	}

	return nil
}

func LogInToAz() error {
	log.Debug("Logging user in to Azure Cli...")
	azCmd := exec.Command("az", "login", "--allow-no-subscriptions")
	loading := make(chan bool)
	go showLoader(loading)
	azCmd.Stdin = os.Stdin
	azCmd.Stdout = os.Stdout
	azCmd.Stderr = os.Stderr
	err := azCmd.Run()
	loading <- true
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
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getSubscriptionIdCmd.CombinedOutput()
	loading <- true
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
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getResourceGroupCmd.CombinedOutput()
	loading <- true
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
	loading := make(chan bool)
	go showLoader(loading)

	_, err := listReposCmd.CombinedOutput()
	loading <- true
	if err != nil {
		log.Fatal("Github repo not found")
		return err
	}
	return nil
}

func AzAppExists(appName string) bool {
	filter := fmt.Sprintf("displayName eq '%s'", appName)
	checkAppExistsCmd := exec.Command("az", "ad", "app", "list", "--only-show-errors", "--filter", filter, "--query", "[].appId")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := checkAppExistsCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return false
	}

	var azApp []string
	json.Unmarshal(out, &azApp)

	return len(azApp) >= 1
}

func (sc *SetUpCmd) ServicePrincipalExists() bool {
	checkSpExistsCmd := exec.Command("az", "ad", "sp", "show", "--only-show-errors", "--id", sc.appId, "--query", "id")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := checkSpExistsCmd.CombinedOutput()
	loading <- true
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
	loading := make(chan bool)
	go showLoader(loading)

	out, err := checkAcrExistsCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return false
	}

	var azAcr []interface{}
	json.Unmarshal(out, &azAcr)

	return len(azAcr) >= 1
}

func AzAksExists(aksName string, resourceGroup string) bool {
	checkAksExistsCmd := exec.Command("az", "aks", "browse", "-g", resourceGroup, "--name", aksName)
	loading := make(chan bool)
	go showLoader(loading)
	_, err := checkAksExistsCmd.CombinedOutput()
	loading <- true

	return err == nil
}

func GetCurrentAzSubscriptionLabel() (SubLabel, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return SubLabel{}, fmt.Errorf("failed to log in to Azure CLI: %v", err)
		}
	}

	getAccountCmd := exec.Command("az", "account", "show", "--query", "{id: id, name: name}")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getAccountCmd.CombinedOutput()
	loading <- true
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
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getAccountCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return nil, fmt.Errorf("get subscription labels: %w", err)
	}

	var subLabels []SubLabel
	if err := json.Unmarshal(out, &subLabels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscription labels: %w", err)
	} else if len(subLabels) == 0 {
		return nil, errors.New("no subscriptions found")
	}

	return subLabels, nil
}

func GetAzResourceGroups() ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	getResourceGroupsCmd := exec.Command("az", "group", "list", "--query", "[].name")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getResourceGroupsCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return nil, fmt.Errorf("failed to get resource groups: %w", err)
	}

	var resourceGroups []string
	if err := json.Unmarshal(out, &resourceGroups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource groups: %w", err)
	}

	return resourceGroups, nil
}

func GetAzContainerRegistries(resourceGroup string) ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	getAcrCmd := exec.Command("az", "acr", "list", "-g", resourceGroup, "--query", "[].name")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getAcrCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return nil, fmt.Errorf("failed to get container registries: %w", err)
	}

	var acrs []string
	if err := json.Unmarshal(out, &acrs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container registries: %w", err)
	}

	return acrs, nil
}

func GetAzClusters(clusterResourceGroup string) ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	getClustersCmd := exec.Command("az", "aks", "list", "-g", clusterResourceGroup, "--query", "[].name")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getClustersCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return nil, fmt.Errorf("failed to get clusters: %w", err)
	}

	var clusters []string
	if err := json.Unmarshal(out, &clusters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clusters: %w", err)
	}

	return clusters, nil
}

func GetAzNamespaces(clusterResourceGroup, clusterName string) ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	err := checkKubectlInstalled(clusterResourceGroup, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to check if kubectl is properly installed: %w", err)
	}

	getNamespacesCmd := exec.Command("az", "aks", "command", "invoke", "-g", clusterResourceGroup, "-n", clusterName, "--command", "kubectl get namespaces -o json", "-o", "json")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getNamespacesCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	var jsonData struct {
		LogData string `json:"logs"`
	}

	if err := json.Unmarshal(out, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal namespaces: %w", err)
	}

	type Metadata struct {
		Name string `json:"name"`
	}

	type Items struct {
		Meta Metadata `json:"metadata"`
	}

	var log struct {
		Namespaces []Items `json:"items"`
	}

	if err := json.Unmarshal([]byte(jsonData.LogData), &log); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs data: %w", err)
	} else if len(log.Namespaces) == 0 {
		return nil, errors.New("no namespaces found")
	}

	namespaces := make([]string, len(log.Namespaces))
	for i, item := range log.Namespaces {
		namespaces[i] = item.Meta.Name
	}

	return namespaces, nil
}

func GetAzLocations() ([]string, error) {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return nil, fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	getLocationsCmd := exec.Command("az", "account", "list-locations", "--query", "[].name")
	loading := make(chan bool)
	go showLoader(loading)

	out, err := getLocationsCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return nil, fmt.Errorf("failed to get locations: %w", err)
	}

	var locations []string
	if err := json.Unmarshal(out, &locations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal locations: %w", err)
	} else if len(locations) == 0 {
		return nil, errors.New("no locations found")
	}

	return locations, nil
}

func CreateAzResourceGroup(resourceGroup, location string) error {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	createResourceGroupCmd := exec.Command("az", "group", "create", "--name", resourceGroup, "--location", location)
	loading := make(chan bool)
	go showLoader(loading)

	_, err := createResourceGroupCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return fmt.Errorf("failed to create resource group %s: %w", resourceGroup, err)
	}

	return nil
}

func CreateAzContainerRegistry(acr, resourceGroup, sku string) error {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	createAcrCmd := exec.Command("az", "acr", "create", "--name", acr, "--resource-group", resourceGroup, "--sku", sku)
	loading := make(chan bool)
	go showLoader(loading)

	_, err := createAcrCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return fmt.Errorf("failed to create container registry %s: %w", acr, err)
	}

	return nil
}

func CreateAzCluster(cluster, resourceGroup, privacySetting string) error {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	var createClusterCmd *exec.Cmd

	switch privacySetting {
	case "public":
		createClusterCmd = exec.Command("az", "aks", "create", "--name", cluster, "--resource-group", resourceGroup)
	case "private":
		createClusterCmd = exec.Command("az", "aks", "create", "--name", cluster, "--resource-group", resourceGroup, "--enable-private-cluster")
	}

	loading := make(chan bool)
	go showLoader(loading)

	_, err := createClusterCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return fmt.Errorf("failed to create cluster %s: %w", cluster, err)
	}

	return nil
}

func AttachAcrToCluster(cluster, resourceGroup, acr string) error {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	attachAcrCmd := exec.Command("az", "aks", "update", "--name", cluster, "--resource-Group", resourceGroup, "--attach-acr", acr)
	loading := make(chan bool)
	go showLoader(loading)

	_, err := attachAcrCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return fmt.Errorf("failed to attach acr %s to cluster %s: %w", acr, cluster, err)
	}

	return nil
}

func CreateAzNamespace(namespace, resourceGroup, clusterName string) error {
	CheckAzCliInstalled()
	if !IsLoggedInToAz() {
		if err := LogInToAz(); err != nil {
			return fmt.Errorf("failed to log in to Azure CLI: %w", err)
		}
	}

	createNamespaceCmd := exec.Command("az", "aks", "command", "invoke", "-g", resourceGroup, "-n", clusterName, "--command", "kubectl create namespace "+namespace)
	loading := make(chan bool)
	go showLoader(loading)

	_, err := createNamespaceCmd.CombinedOutput()
	loading <- true
	if err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
	}

	return nil
}

func showLoader(loading chan bool) {
	loader := []string{"running... |", "running... /", "running... -", "running... \\"}
	i := 0
	for {
		select {
		case <-loading:
			fmt.Print("\r")
			return
		default:
			fmt.Printf("\r%s", loader[i%len(loader)])
			time.Sleep(100 * time.Millisecond)
			i++
		}
	}
}
