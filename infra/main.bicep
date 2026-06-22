@description('The name of your VM')
param vmName string = 'hurion-bot'

@description('Username for the Virtual Machine')
param adminUsername string = 'azureuser'

@description('SSH Public Key for the Virtual Machine')
@secure()
param adminPasswordOrKey string

@description('The size of the VM')
param vmSize string = 'Standard_B1s'

@description('The name of the Key Vault where secrets are stored')
param keyVaultName string = 'hurion-${uniqueString(resourceGroup().id)}'

@description('The Docker Image to run (e.g., jorge/hurion-bot:latest)')
param dockerImage string = 'jorge/hurion-bot:latest'

@description('The content of the .env file')
@secure()
param appEnvContent string

@description('The content of the .docker.env file (Optional)')
@secure()
param dockerEnvContent string = ''

var composeProdYaml = loadTextContent('compose.yaml')

var configYaml = loadTextContent('../config.yaml')

@description('A unique integer to trigger re-deployment (e.g., a timestamp)')
param deployTimestamp int = dateTimeToEpoch(utcNow())

@description('Location for all resources')
param location string = resourceGroup().location

var publicIPAddressName = '${vmName}PublicIP'
var networkInterfaceName = '${vmName}NetInt'
var osDiskType = 'Standard_LRS'
var virtualNetworkName = '${vmName}VNet'
var subnetName = 'Subnet'
var networkSecurityGroupName = '${vmName}NSG'

// 1. Key Vault for Secrets
resource vault 'Microsoft.KeyVault/vaults@2022-07-01' = {
  name: keyVaultName
  location: location
  properties: {
    sku: {
      family: 'A'
      name: 'standard'
    }
    tenantId: subscription().tenantId
    enableRbacAuthorization: true 
    accessPolicies: []
  }
}

// 1.1 Store Secrets in Key Vault securely
resource appEnvSecret 'Microsoft.KeyVault/vaults/secrets@2022-07-01' = {
  parent: vault
  name: 'APP-ENV'
  properties: {
    value: appEnvContent
  }
}

resource dockerEnvSecret 'Microsoft.KeyVault/vaults/secrets@2022-07-01' = if (!empty(dockerEnvContent)) {
  parent: vault
  name: 'DOCKER-ENV'
  properties: {
    value: dockerEnvContent
  }
}

// 2. Networking
resource publicIP 'Microsoft.Network/publicIPAddresses@2024-03-01' = {
  name: publicIPAddressName
  location: location
  sku: {
    name: 'Standard'
  }
  properties: {
    publicIPAllocationMethod: 'Static'
  }
}

resource nsg 'Microsoft.Network/networkSecurityGroups@2024-03-01' = {
  name: networkSecurityGroupName
  location: location
  properties: {
    securityRules: [
      {
        name: 'SSH'
        properties: {
          priority: 1000
          protocol: 'Tcp'
          access: 'Allow'
          direction: 'Inbound'
          sourceAddressPrefix: '*'
          sourcePortRange: '*'
          destinationAddressPrefix: '*'
          destinationPortRange: '22'
        }
      }
    ]
  }
}

resource vnet 'Microsoft.Network/virtualNetworks@2024-03-01' = {
  name: virtualNetworkName
  location: location
  properties: {
    addressSpace: {
      addressPrefixes: [
        '10.0.0.0/16'
      ]
    }
    subnets: [
      {
        name: subnetName
        properties: {
          addressPrefix: '10.0.0.0/24'
          networkSecurityGroup: {
            id: nsg.id
          }
        }
      }
    ]
  }
}

resource nic 'Microsoft.Network/networkInterfaces@2024-03-01' = {
  name: networkInterfaceName
  location: location
  properties: {
    ipConfigurations: [
      {
        name: 'ipconfig1'
        properties: {
          subnet: {
            id: vnet.properties.subnets[0].id
          }
          privateIPAllocationMethod: 'Dynamic'
          publicIPAddress: {
            id: publicIP.id
          }
        }
      }
    ]
  }
}

// 3. Virtual Machine with Managed Identity
resource vm 'Microsoft.Compute/virtualMachines@2024-03-01' = {
  name: vmName
  location: location
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    hardwareProfile: {
      vmSize: vmSize
    }
    storageProfile: {
      osDisk: {
        createOption: 'FromImage'
        managedDisk: {
          storageAccountType: osDiskType
        }
      }
      imageReference: {
        publisher: 'Canonical'
        offer: '0001-com-ubuntu-server-jammy'
        sku: '22_04-lts-gen2'
        version: 'latest'
      }
    }
    networkProfile: {
      networkInterfaces: [
        {
          id: nic.id
        }
      ]
    }
    osProfile: {
      computerName: vmName
      adminUsername: adminUsername
      linuxConfiguration: {
        disablePasswordAuthentication: true
        ssh: {
          publicKeys: [
            {
              path: '/home/${adminUsername}/.ssh/authorized_keys'
              keyData: adminPasswordOrKey
            }
          ]
        }
      }
    }
  }
}

// 4. Grant VM access to the Key Vault
resource roleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  name: guid(vault.id, vm.id, 'KeyVaultSecretsUser')
  scope: vault
  properties: {
    roleDefinitionId: subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '4633458b-17de-408a-b874-0445c86b69e6') 
    principalId: vm.identity.principalId
    principalType: 'ServicePrincipal'
  }
}

// 5. Custom Script Extension (The "Remote Update" Engine)
var rawScript = '''#!/bin/bash
set -e

# Step 1: Install Docker and Azure CLI dependencies
install_docker() {
    if ! command -v docker &> /dev/null; then
        apt-get update
        apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
        apt-get update
        apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
        usermod -aG docker {{adminUsername}}
    fi

    if ! command -v az &> /dev/null; then
        curl -sL https://aka.ms/InstallAzureCLIDeb | bash
    fi
}

# Step 2: Write all Configuration files (Pushed from Local/Bicep)
write_configs() {
    mkdir -p /hurion/app

    cat <<'EOF' > /hurion/app/compose.yaml
{{composeProdYaml}}
EOF

    cat <<'EOF' > /hurion/app/config.yaml
{{configYaml}}
EOF
}

# Step 3: Fetch app secrets and Docker Hub environment from Key Vault
pull_secrets() {
    az login --identity
    az keyvault secret show --vault-name {{keyVaultName}} --name "APP-ENV" --query "value" -o tsv > /hurion/app/.env
    echo "" >> /hurion/app/.env
    echo "IMAGE_NAME={{dockerImage}}" >> /hurion/app/.env
}

# Step 4: Login to private Docker Hub registry if credentials exist
docker_login() {
    if az keyvault secret show --vault-name {{keyVaultName}} --name "DOCKER-ENV" &>/dev/null; then
        az keyvault secret show --vault-name {{keyVaultName}} --name "DOCKER-ENV" --query "value" -o tsv > /hurion/app/.docker.env
        
        local user=$(grep -E "^DOCKER_USER=" /hurion/app/.docker.env | cut -d"=" -f2-)
        local pat=$(grep -E "^DOCKER_PAT=" /hurion/app/.docker.env | cut -d"=" -f2-)
        
        if [ -n "$user" ] && [ -n "$pat" ]; then
            echo "$pat" | docker login -u "$user" --password-stdin
        fi
    fi
}

# Step 5: Start / Restart the containerized bot
run_application() {
    cd /hurion/app
    mkdir -p /hurion/data
    chown -R {{adminUsername}}:{{adminUsername}} /hurion
    
    export IMAGE_NAME="{{dockerImage}}"
    docker compose pull
    docker compose up -d
}

# Execute modular steps
install_docker
write_configs
pull_secrets
docker_login
run_application
'''

var script1 = replace(rawScript, '{{adminUsername}}', adminUsername)
var script2 = replace(script1, '{{composeProdYaml}}', composeProdYaml)
var script3 = replace(script2, '{{configYaml}}', configYaml)
var script4 = replace(script3, '{{keyVaultName}}', keyVaultName)
var finalScript = replace(script4, '{{dockerImage}}', dockerImage)

resource updateScript 'Microsoft.Compute/virtualMachines/extensions@2024-03-01' = {
  parent: vm
  name: 'UpdateHurion'
  location: location
  dependsOn: [
    appEnvSecret
    dockerEnvSecret
    roleAssignment
  ]
  properties: {
    publisher: 'Microsoft.Azure.Extensions'
    type: 'CustomScript'
    typeHandlerVersion: '2.1'
    autoUpgradeMinorVersion: true
    settings: {
      timestamp: deployTimestamp 
    }
    protectedSettings: {
      script: base64(finalScript)
    }
  }
}

output publicIP string = publicIP.properties.ipAddress
output keyVaultName string = vault.name
