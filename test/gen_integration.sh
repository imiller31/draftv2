export WORKFLOWS_PATH=.github/workflows

# remove previous tests
echo "Removing previous integration configs"
rm -rf ./integration/*
echo "Removing previous integration workflows"
rm ../$WORKFLOWS_PATH/integration-linux.yml
rm ../$WORKFLOWS_PATH/integration-windows.yml

# create temp files for keeping track off workflow jobs to build job-dependency graph
# this is used to populated the needs: field of the required final workflow jobs
helm_workflow_names_file=./temp/helm_workflow_names.txt
rm $helm_workflow_names_file
helm_win_workflow_names_file=./temp/helm_win_workflow_names.txt
rm $helm_win_workflow_names_file
kustomize_workflow_names_file=./temp/kustomize_workflow_names.txt
rm $kustomize_workflow_names_file
kustomize_win_workflow_names_file=./temp/kustomize_win_workflow_names.txt
rm $kustomize_win_workflow_names_file
manifest_workflow_names_file=./temp/manifest_workflow_names.txt
rm $manifest_workflow_names_file
mkdir -p ./temp

# add build to workflow
echo "# this file is generated using gen_integration.sh
name: draft Linux Integrations
on:
  pull_request:
    branches: [ main, staging ]
  workflow_dispatch:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.22
      - name: make
        run: make
      - uses: actions/upload-artifact@v3
        with:
          name: helm-skaffold
          path: ./test/skaffold.yaml
          if-no-files-found: error
      - uses: actions/upload-artifact@v3
        with:
          name: draft-binary
          path: ./draft
          if-no-files-found: error" > ../$WORKFLOWS_PATH/integration-linux.yml

echo "name: draft Windows Integrations
on:
  pull_request:
    branches: [ main, staging ]
  workflow_dispatch:
jobs:
  build:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.22
      - name: make
        run: make
      - uses: actions/upload-artifact@v3
        with:
          name: draft-binary
          path: ./draft.exe
          if-no-files-found: error
      - uses: actions/upload-artifact@v3
        with:
          name: check_windows_helm
          path: ./test/check_windows_helm.ps1
          if-no-files-found: error
      - uses: actions/upload-artifact@v3
        with:
          name: check_windows_addon_helm
          path: ./test/check_windows_addon_helm.ps1
          if-no-files-found: error
      - uses: actions/upload-artifact@v3
        with:
          name: check_windows_kustomize
          path: ./test/check_windows_kustomize.ps1
          if-no-files-found: error
      - uses: actions/upload-artifact@v3
        with:
          name: check_windows_addon_kustomize
          path: ./test/check_windows_addon_kustomize.ps1
          if-no-files-found: error" > ../$WORKFLOWS_PATH/integration-windows.yml


# read config and add integration test for each language
cat integration_config.json | jq -c '.[]' | while read -r test; 
do 
    note="# this file is generated using gen_integration.sh"
    # extract from json
    lang=$(echo $test | jq '.language' -r)
    version=$(echo $test | jq '.version' -r)
    builderversion=$(echo $test | jq '.builderversion' -r)
    port=$(echo $test | jq '.port' -r)
    serviceport=$(echo $test | jq '.serviceport' -r)
    repo=$(echo $test | jq '.repo' -r)

    imagename="host.minikube.internal:5001/testapp"
    # addon integration testing vars
    ingress_test_args="-a webapp_routing --variable ingress-tls-cert-keyvault-uri=test.cert.keyvault.uri --variable ingress-use-osm-mtls=true --variable ingress-host=host1"
    create_config_args="--variable PORT=8080 --variable APPNAME=testingCreateCommand --variable VERSION=1.11 --variable BUILDERVERSION=1.11 --variable SERVICEPORT=8080 --variable NAMESPACE=testNamespace --variable IMAGENAME=testImage --variable IMAGETAG=latest"
    python_create_config_args="--variable PORT=8080 --variable APPNAME=testingCreateCommand --variable VERSION=1.11 --variable BUILDERVERSION=1.11 --variable SERVICEPORT=8080 --variable NAMESPACE=testNamespace --variable IMAGENAME=testImage --variable IMAGETAG=latest --variable ENTRYPOINT=testapp.py"
    echo "Adding $lang with port $port"

    mkdir ./integration/$lang

    # create helm.yaml
    echo "$note
deployType: \"Helm\"
languageType: \"$lang\"
deployVariables:
  - name: \"PORT\"
    value: \"$port\"
  - name: \"SERVICEPORT\"
    value: \"$serviceport\"
  - name: \"APPNAME\"
    value: \"testapp\"
  - name: \"IMAGENAME\"
    value: \"$imagename\"
languageVariables:
  - name: \"VERSION\"
    value: \"$version\"
  - name: \"BUILDERVERSION\"
    value: \"$builderversion\"
  - name: \"PORT\"
    value: \"$port\"" > ./integration/$lang/helm.yaml

    # create kustomize.yaml
    echo "$note
deployType: \"kustomize\"
languageType: \"$lang\"
deployVariables:
  - name: \"PORT\"
    value: \"$port\"
  - name: \"SERVICEPORT\"
    value: \"$serviceport\"
  - name: \"APPNAME\"
    value: \"testapp\"
  - name: \"IMAGENAME\"
    value: \"$imagename\"
languageVariables:
  - name: \"VERSION\"
    value: \"$version\"
  - name: \"BUILDERVERSION\"
    value: \"$builderversion\"
  - name: \"PORT\"
    value: \"$port\"" > ./integration/$lang/kustomize.yaml

    # create manifest.yaml
    echo "$note
deployType: \"manifests\"
languageType: \"$lang\"
deployVariables:
  - name: \"PORT\"
    value: \"$port\"
  - name: \"SERVICEPORT\"
    value: \"$serviceport\"
  - name: \"APPNAME\"
    value: \"testapp\"
  - name: \"IMAGENAME\"
    value: \"$imagename\"
languageVariables:
  - name: \"VERSION\"
    value: \"$version\"
  - name: \"BUILDERVERSION\"
    value: \"$builderversion\"
  - name: \"PORT\"
    value: \"$port\"" > ./integration/$lang/manifest.yaml

    # create helm workflow
    helm_create_update_job_name=$lang-helm-create-update
    echo $helm_create_update_job_name >> $helm_workflow_names_file
    echo "
  $lang-helm-dry-run:
      runs-on: ubuntu-latest
      needs: build
      steps:
        - uses: actions/checkout@v3
        - uses: actions/download-artifact@v3
          with:
            name: draft-binary
        - run: chmod +x ./draft
        - run: mkdir ./langtest
        - uses: actions/checkout@v3
          with:
            repository: $repo
            path: ./langtest
        - name: Execute Dry Run with config file
          run: |
            mkdir -p test/temp
            ./draft --dry-run --dry-run-file test/temp/dry-run.json \
            create -c ./test/integration/$lang/helm.yaml \
            -d ./langtest/ --skip-file-detection
        - name: Validate JSON
          run: |
            npm install -g ajv-cli@5.0.0
            ajv validate -s test/dry_run_schema.json -d test/temp/dry-run.json
        - name: Execute Dry Run with variables passed through flag
          $(if [ "$lang" = "python" ] 
          then echo "run: |
            mkdir -p test/temp
            ./draft --dry-run --dry-run-file test/temp/dry-run.json \
            create -d ./langtest/ -l $lang --skip-file-detection --deploy-type helm \
            $python_create_config_args"
          else echo "run: |
            mkdir -p test/temp
            ./draft --dry-run --dry-run-file test/temp/dry-run.json \
            create -d ./langtest/ -l $lang --skip-file-detection --deploy-type helm \
            $create_config_args"
            fi)
        - name: Validate JSON
          run: |
            npm install -g ajv-cli@5.0.0
            ajv validate -s test/dry_run_schema.json -d test/temp/dry-run.json
  $helm_create_update_job_name:
    runs-on: ubuntu-latest
    services:
      registry:
        image: registry:2
        ports:
          - 5001:5000
    needs: $lang-helm-dry-run
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - run: chmod +x ./draft
      - run: mkdir ./langtest
      - uses: actions/checkout@v3
        with:
          repository: $repo
          path: ./langtest
      - run: rm -rf ./langtest/manifests && rm -f ./langtest/Dockerfile ./langtest/.dockerignore
      - run: ./draft -v create -c ./test/integration/$lang/helm.yaml -d ./langtest/
      - name: start minikube
        id: minikube
        uses: medyagh/setup-minikube@master
        with:
          insecure-registry: 'host.minikube.internal:5001,10.0.0.0/24'
      # Runs Helm to create manifest files
      - name: Bake deployment
        uses: azure/k8s-bake@v2.2
        with:
          renderEngine: 'helm'
          helmChart: ./langtest/charts
          overrideFiles: ./langtest/charts/values.yaml
          overrides: |
            replicas:2
          helm-version: 'latest'
          releaseName: 'test-release'
        id: bake
      - name: Build and Push image
        continue-on-error: true
        run: |
          export SHELL=/bin/bash
          eval \$(minikube -p minikube docker-env)
          docker build -f ./langtest/Dockerfile -t testapp ./langtest/
          docker tag testapp $imagename
          echo -n \"verifying images:\"
          docker images
          docker push $imagename
          echo 'Curling host.minikube.internal test app images from minikube'
          minikube ssh \"curl http://host.minikube.internal:5001/v2/testapp/tags/list\"
      # Deploys application based on manifest files from previous step
      - name: Deploy application
        uses: Azure/k8s-deploy@v4.0
        continue-on-error: true
        id: deploy
        with:
          action: deploy
          manifests: \${{ steps.bake.outputs.manifestsBundle }}
          images: |
            $imagename
      - name: Wait for rollout
        continue-on-error: true
        id: rollout
        run: |
          kubectl rollout status deployment/test-release-testapp --timeout=2m
      - name: Print K8s Objects
        run: |
          kubectl get po -o json
          kubectl get svc -o json
          kubectl get deploy -o json
      - name: Curl Endpoint
        run: |
          kubectl get svc
          echo 'Starting minikube tunnel'
          minikube tunnel  > /dev/null 2>&1 & tunnelPID=\$!
          sleep 120
          kubectl get svc
          SERVICEIP=\$(kubectl get svc -o jsonpath={'.items[1].status.loadBalancer.ingress[0].ip'})
          echo \"SERVICEIP: \$SERVICEIP\"
          echo 'Curling service IP'
          curl -m 3 \$SERVICEIP:$serviceport
          kill \$tunnelPID
      - run: |
          ./draft -v generate-workflow -d ./langtest/ --deploy-type helm --variable WORKFLOWNAME=someWorkflow --variable BRANCHNAME=main --variable ACRRESOURCEGROUP=someAcrResourceGroup --variable AZURECONTAINERREGISTRY=someRegistry --variable ACRREPOSITORYNAME=someRepository --variable CLUSTERRESOURCEGROUP=someClusterResourceGroup --variable CLUSTERNAME=someAksCluster --variable DOCKERFILEPATH=./Dockerfile --variable BUILDCONTEXTPATH=. --variable NAMESPACE=default --variable PRIVATECLUSTER=false
          pwd
      # Validate generated workflow yaml
      - name: Install action-validator with asdf
        uses: asdf-vm/actions/install@v1
        with:
          tool_versions: |
            action-validator 0.1.2
      - name: Lint Actions
        run: |
          find $WORKFLOWS_PATH -type f \( -iname \*.yaml -o -iname \*.yml \) \
            | xargs -I {} action-validator --verbose {}
      - name: Execute dry run for update command
        run: |
          mkdir -p test/temp
          pwd
          ./draft --dry-run --dry-run-file test/temp/update_dry_run.json update -d ./langtest/ $ingress_test_args  
      - name: Validate JSON
        run: |
          npm install -g ajv-cli@5.0.0
          ajv validate -s test/update_dry_run_schema.json -d test/temp/update_dry_run.json
      - run: ./draft -v update -d ./langtest/ $ingress_test_args
      - name: Check default namespace
        if: steps.deploy.outcome != 'success'
        run: kubectl get po
      - name: Fail if any error
        if: steps.deploy.outcome != 'success'
        run: exit 6" >> ../$WORKFLOWS_PATH/integration-linux.yml

    # create kustomize workflow
    kustomize_create_update_job_name=$lang-kustomize-create-update
    echo $kustomize_create_update_job_name >> $kustomize_workflow_names_file
    echo "
  $lang-kustomize-dry-run:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - run: chmod +x ./draft
      - run: mkdir ./langtest
      - uses: actions/checkout@v3
        with:
          repository: $repo
          path: ./langtest
      - name: Execute Dry Run with config file
        run: |
          mkdir -p test/temp
          ./draft --dry-run --dry-run-file test/temp/dry-run.json \
          create -c ./test/integration/$lang/kustomize.yaml \
          -d ./langtest/ --skip-file-detection
      - name: Validate JSON
        run: |
          npm install -g ajv-cli@5.0.0
          ajv validate -s test/dry_run_schema.json -d test/temp/dry-run.json
      - name: Execute Dry Run with variables passed through flag 
        $(if [ "$lang" = "python" ]; 
        then echo "run: |
          mkdir -p test/temp
          ./draft --dry-run --dry-run-file test/temp/dry-run.json \
          create -d ./langtest/ -l $lang --skip-file-detection --deploy-type kustomize \
          $python_create_config_args";
        else echo "run: |
          mkdir -p test/temp
          ./draft --dry-run --dry-run-file test/temp/dry-run.json \
          create -d ./langtest/ -l $lang --skip-file-detection --deploy-type kustomize \
          $create_config_args";
        fi)
      - name: Validate JSON
        run: |
          npm install -g ajv-cli@5.0.0
          ajv validate -s test/dry_run_schema.json -d test/temp/dry-run.json
  $kustomize_create_update_job_name:
    runs-on: ubuntu-latest
    services:
      registry:
        image: registry:2
        ports:
          - 5001:5000
    needs: $lang-kustomize-dry-run
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - run: chmod +x ./draft
      - run: mkdir ./langtest
      - uses: actions/checkout@v3
        with:
          repository: $repo
          path: ./langtest
      - run: rm -rf ./langtest/manifests && rm -f ./langtest/Dockerfile ./langtest/.dockerignore
      - run: ./draft -v create -c ./test/integration/$lang/kustomize.yaml -d ./langtest/
      - name: start minikube
        id: minikube
        uses: medyagh/setup-minikube@master
        with:
          insecure-registry: 'host.minikube.internal:5001,10.0.0.0/24'
      - name: Bake deployment
        uses: azure/k8s-bake@v2.4
        id: bake
        with:
          renderEngine: 'kustomize'
          kustomizationPath: ./langtest/base
          kubectl-version: 'latest'
      - name: Build and Push Image
        continue-on-error: true
        run: |
          eval \$(minikube docker-env)
          docker build -f ./langtest/Dockerfile -t testapp ./langtest/
          docker tag testapp $imagename
          echo -n \"verifying images:\"
          docker images
          docker push $imagename
          echo 'Curling host.minikube.internal test app images from minikube'
          minikube ssh \"curl http://host.minikube.internal:5001/v2/testapp/tags/list\"
      # Deploys application based on manifest files from previous step
      - name: Deploy application
        uses: Azure/k8s-deploy@v4.0
        continue-on-error: true
        id: deploy
        with:
          action: deploy
          manifests: \${{ steps.bake.outputs.manifestsBundle }}
          images: |
            $imagename
      - name: Wait for rollout
        continue-on-error: true
        id: rollout
        run: |
          kubectl rollout status deployment/testapp --timeout=2m
      - name: Print K8s Objects
        run: |
          kubectl get po -o json
          kubectl get svc -o json
          kubectl get deploy -o json
      - name: Curl Endpoint
        run: |
          kubectl get svc
          echo 'Starting minikube tunnel'
          minikube tunnel  > /dev/null 2>&1 & tunnelPID=\$!
          sleep 120
          kubectl get svc
          SERVICEIP=\$(kubectl get svc -o jsonpath={'.items[1].status.loadBalancer.ingress[0].ip'})
          echo \"SERVICEIP: \$SERVICEIP\"
          echo 'Curling service IP'
          curl -m 3 \$SERVICEIP:$serviceport
          kill \$tunnelPID
      - run: ./draft -v generate-workflow -d ./langtest/ --deploy-type kustomize --variable WORKFLOWNAME=someWorkflow --variable BRANCHNAME=main --variable ACRRESOURCEGROUP=someAcrResourceGroup --variable AZURECONTAINERREGISTRY=someRegistry --variable ACRREPOSITORYNAME=someRepository --variable CLUSTERRESOURCEGROUP=someClusterResourceGroup --variable CLUSTERNAME=someAksCluster --variable DOCKERFILEPATH=./Dockerfile --variable BUILDCONTEXTPATH=. --variable NAMESPACE=default --variable PRIVATECLUSTER=false
      # Validate generated workflow yaml
      - name: Install action-validator with asdf
        uses: asdf-vm/actions/install@v1
        with:
          tool_versions: |
            action-validator 0.1.2
      - name: Lint Actions
        run: |
          find $WORKFLOWS_PATH -type f \( -iname \*.yaml -o -iname \*.yml \) \
            | xargs -I {} action-validator --verbose {}
      - name: Execute dry run for update command
        run: |
          mkdir -p test/temp
          ./draft --dry-run --dry-run-file test/temp/update_dry_run.json update -d ./langtest/ $ingress_test_args  
      - name: Validate JSON
        run: |
          npm install -g ajv-cli@5.0.0
          ajv validate -s test/update_dry_run_schema.json -d test/temp/update_dry_run.json
      - run: ./draft -v update -d ./langtest/ $ingress_test_args
      - name: Check default namespace
        if: steps.deploy.outcome != 'success'
        run: kubectl get po
      - name: Fail if any error
        if: steps.deploy.outcome != 'success'
        run: exit 6" >> ../$WORKFLOWS_PATH/integration-linux.yml

  # create manifests workflow
    manifest_update_job_name=$lang-manifest-update
    echo $manifest_update_job_name >> $manifest_workflow_names_file
    echo "
  $lang-manifest-dry-run:
      runs-on: ubuntu-latest
      needs: build
      steps:
        - uses: actions/checkout@v3
        - uses: actions/download-artifact@v3
          with:
            name: draft-binary
        - run: chmod +x ./draft
        - run: mkdir ./langtest
        - uses: actions/checkout@v3
          with:
            repository: $repo
            path: ./langtest
        - name: Execute Dry Run with config file
          run: |
            mkdir -p test/temp
            ./draft --dry-run --dry-run-file test/temp/dry-run.json \
            create -c ./test/integration/$lang/manifest.yaml \
            -d ./langtest/ --skip-file-detection
        - name: Validate JSON
          run: |
            npm install -g ajv-cli@5.0.0
            ajv validate -s test/dry_run_schema.json -d test/temp/dry-run.json
        - name: Execute Dry Run with variables passed through flag
          $(if [ "$lang" = "python" ]; 
          then echo "run: |
            mkdir -p test/temp
            ./draft --dry-run --dry-run-file test/temp/dry-run.json \
            create -d ./langtest/ -l $lang --skip-file-detection --deploy-type manifests \
            $python_create_config_args";
          else echo "run: |
            mkdir -p test/temp
            ./draft --dry-run --dry-run-file test/temp/dry-run.json \
            create -d ./langtest/ -l $lang --skip-file-detection --deploy-type manifests \
            $create_config_args";
          fi) 
        - name: Validate JSON
          run: |
            npm install -g ajv-cli@5.0.0
            ajv validate -s test/dry_run_schema.json -d test/temp/dry-run.json
  $lang-manifests-create:
    runs-on: ubuntu-latest
    services:
      registry:
        image: registry:2
        ports:
          - 5001:5000
    needs: $lang-manifest-dry-run
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - run: chmod +x ./draft
      - run: mkdir ./langtest
      - uses: actions/checkout@v3
        with:
          repository: $repo
          path: ./langtest
      - run: rm -rf ./langtest/manifests && rm -f ./langtest/Dockerfile ./langtest/.dockerignore
      - run: ./draft -v create -c ./test/integration/$lang/manifest.yaml -d ./langtest/
      - name: print manifests
        run: cat ./langtest/manifests/*
      - name: Add docker.local host to /etc/hosts
        run: |
          sudo echo \"127.0.0.1 docker.local\" | sudo tee -a /etc/hosts
      - name: start minikube
        id: minikube
        uses: medyagh/setup-minikube@master
        with:
          insecure-registry: 'host.minikube.internal:5001,10.0.0.0/24'
      - name: Build and Push Image
        continue-on-error: true
        run: |
          eval \$(minikube docker-env)
          docker build -f ./langtest/Dockerfile -t testapp ./langtest/
          docker tag testapp $imagename
          echo -n \"verifying images:\"
          docker images
          docker push $imagename
          echo 'Curling host.minikube.internal test app images from minikube'
          minikube ssh \"curl http://host.minikube.internal:5001/v2/testapp/tags/list\"
      # Deploys application based on manifest files from previous step
      - name: Deploy application
        run: kubectl apply -f ./langtest/manifests/
        continue-on-error: true
        id: deploy
      - name: Wait for rollout
        continue-on-error: true
        id: rollout
        run: |
          kubectl rollout status deployment/testapp --timeout=2m
      - name: Print K8s Objects
        run: |
          kubectl get po -o json
          kubectl get svc -o json
          kubectl get deploy -o json
      - name: Curl Endpoint
        run: |
          kubectl get svc
          echo 'Starting minikube tunnel'
          minikube tunnel  > /dev/null 2>&1 & tunnelPID=\$!
          sleep 120
          kubectl get svc
          SERVICEIP=\$(kubectl get svc -o jsonpath={'.items[1].status.loadBalancer.ingress[0].ip'})
          echo \"SERVICEIP: \$SERVICEIP\"
          echo 'Curling service IP'
          curl -m 3 \$SERVICEIP:$serviceport
          kill \$tunnelPID
      - run: ./draft -v generate-workflow -d ./langtest/ --deploy-type manifests --variable WORKFLOWNAME=someWorkflow --variable BRANCHNAME=main --variable ACRRESOURCEGROUP=someAcrResourceGroup --variable AZURECONTAINERREGISTRY=someRegistry --variable ACRREPOSITORYNAME=someRepository --variable CLUSTERRESOURCEGROUP=someClusterResourceGroup --variable CLUSTERNAME=someAksCluster --variable DOCKERFILEPATH=./Dockerfile --variable BUILDCONTEXTPATH=. --variable NAMESPACE=default --variable PRIVATECLUSTER=false
      # Validate generated workflow yaml
      - name: Install action-validator with asdf
        uses: asdf-vm/actions/install@v1
        with:
          tool_versions: |
            action-validator 0.1.2
      - name: Lint Actions
        run: |
          find $WORKFLOWS_PATH -type f \( -iname \*.yaml -o -iname \*.yml \) \
            | xargs -I {} action-validator --verbose {}
      - uses: actions/upload-artifact@v3
        with:
          name: $lang-manifests-create
          path: ./langtest
      - name: Fail if any error
        if: steps.deploy.outcome != 'success' || steps.rollout.outcome != 'success'
        run: exit 6
  $manifest_update_job_name:
    needs: $lang-manifests-create
    runs-on: ubuntu-latest
    services:
      registry:
        image: registry:2
        ports:
          - 5000:5000
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - run: chmod +x ./draft
      - uses: actions/download-artifact@v3
        with:
          name: $lang-manifests-create
          path: ./langtest/
      - name: Execute dry run for update command
        run: |
          mkdir -p test/temp
          ./draft --dry-run --dry-run-file test/temp/update_dry_run.json update -d ./langtest/ $ingress_test_args  
      - name: Validate JSON
        run: |
          npm install -g ajv-cli@5.0.0
          ajv validate -s test/update_dry_run_schema.json -d test/temp/update_dry_run.json
      - run: ./draft -v update -d ./langtest/ $ingress_test_args
      - name: start minikube
        id: minikube
        uses: medyagh/setup-minikube@master
      - name: Build image
        run: |
          export SHELL=/bin/bash
          eval \$(minikube -p minikube docker-env)
          docker build -f ./langtest/Dockerfile -t testapp ./langtest/
          echo -n "verifying images:"
          docker images
      # Deploys application based on manifest files from previous step
      - name: Deploy application
        run: kubectl apply -f ./langtest/manifests/
        continue-on-error: true
        id: deploy
      - name: Check default namespace
        if: steps.deploy.outcome != 'success'
        run: kubectl get po
      - name: Fail if any error
        if: steps.deploy.outcome != 'success'
        run: exit 6" >> ../$WORKFLOWS_PATH/integration-linux.yml

  helm_update_win_jobname=$lang-helm-update
  echo $helm_update_win_jobname >> $helm_win_workflow_names_file
    # create helm workflow
    echo "
  $lang-helm-create:
    runs-on: windows-latest
    needs: build
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - run: mkdir ./langtest
      - uses: actions/checkout@v3
        with:
          repository: $repo
          path: ./langtest
      - run: Remove-Item ./langtest/manifests -Recurse -Force -ErrorAction Ignore
      - run: Remove-Item ./langtest/Dockerfile -ErrorAction Ignore
      - run: Remove-Item ./langtest/.dockerignore -ErrorAction Ignore
      - run: ./draft.exe -v create -c ./test/integration/$lang/helm.yaml -d ./langtest/
      - uses: actions/download-artifact@v3
        with:
          name: check_windows_helm
          path: ./langtest/
      - run: ./check_windows_helm.ps1
        working-directory: ./langtest/
      - uses: actions/upload-artifact@v3
        with:
          name: $lang-helm-create
          path: ./langtest
  $helm_update_win_jobname:
    needs: $lang-helm-create
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - uses: actions/download-artifact@v3
        with:
          name: $lang-helm-create
          path: ./langtest/
      - run: Remove-Item ./langtest/charts/templates/ingress.yaml -Recurse -Force -ErrorAction Ignore
      - run: ./draft.exe -v update -d ./langtest/ $ingress_test_args
      - uses: actions/download-artifact@v3
        with:
          name: check_windows_addon_helm
          path: ./langtest/
      - run: ./check_windows_addon_helm.ps1
        working-directory: ./langtest/" >> ../$WORKFLOWS_PATH/integration-windows.yml

    # create kustomize workflow
    kustomize_win_workflow_name=$lang-kustomize-update
    echo $kustomize_win_workflow_name >> $kustomize_win_workflow_names_file
    echo "
  $lang-kustomize-create:
    runs-on: windows-latest
    needs: build
    steps:
      - uses: actions/checkout@v3
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - run: mkdir ./langtest
      - uses: actions/checkout@v3
        with:
          repository: $repo
          path: ./langtest
      - run: Remove-Item ./langtest/manifests -Recurse -Force -ErrorAction Ignore
      - run: Remove-Item ./langtest/Dockerfile -ErrorAction Ignore
      - run: Remove-Item ./langtest/.dockerignore -ErrorAction Ignore
      - run: ./draft.exe -v create -c ./test/integration/$lang/kustomize.yaml -d ./langtest/
      - uses: actions/download-artifact@v3
        with:
          name: check_windows_kustomize
          path: ./langtest/
      - run: ./check_windows_kustomize.ps1
        working-directory: ./langtest/
      - uses: actions/upload-artifact@v3
        with:
          name: $lang-kustomize-create
          path: ./langtest
  $kustomize_win_workflow_name:
    needs: $lang-kustomize-create 
    runs-on: windows-latest
    steps:
      - uses: actions/download-artifact@v3
        with:
          name: draft-binary
      - uses: actions/download-artifact@v3
        with:
          name: $lang-kustomize-create
          path: ./langtest
      - run: Remove-Item ./langtest/overlays/production/ingress.yaml -ErrorAction Ignore
      - run: ./draft.exe -v update -d ./langtest/ $ingress_test_args
      - uses: actions/download-artifact@v3
        with:
          name: check_windows_addon_kustomize
          path: ./langtest/
      - run: ./check_windows_addon_kustomize.ps1
        working-directory: ./langtest/
      " >> ../$WORKFLOWS_PATH/integration-windows.yml
done

echo "
  helm-win-integrations-summary:
      runs-on: windows-latest
      needs: [ $( paste -sd ',' $helm_win_workflow_names_file) ]
      steps:
        - run: echo "helm integrations passed"
" >> ../$WORKFLOWS_PATH/integration-windows.yml

echo "
  kustomize-win-integrations-summary:
      runs-on: windows-latest
      needs: [ $( paste -sd ',' $kustomize_win_workflow_names_file) ]
      steps:
        - run: echo "kustomize integrations passed"
" >> ../$WORKFLOWS_PATH/integration-windows.yml

echo "
  helm-integrations-summary:
      runs-on: ubuntu-latest
      needs: [ $( paste -sd ',' $helm_workflow_names_file) ]
      steps:
        - run: echo "helm integrations passed"
" >> ../$WORKFLOWS_PATH/integration-linux.yml

echo "
  kustomize-integrations-summary:
      runs-on: ubuntu-latest
      needs: [ $( paste -sd ',' $kustomize_workflow_names_file) ]
      steps:
        - run: echo "kustomize integrations passed"
" >> ../$WORKFLOWS_PATH/integration-linux.yml

echo "
  manifest-integrations-summary:
      runs-on: ubuntu-latest
      needs: [ $( paste -sd ',' $manifest_workflow_names_file) ]
      steps:
        - run: echo "manifest integrations passed"
" >> ../$WORKFLOWS_PATH/integration-linux.yml