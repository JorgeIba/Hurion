@description('The name of your VM')
param vmName string = 'hurion-bot'

@description('Username for the Virtual Machine')
param adminUsername string = 'azureuser'

@description('SSH Public Key for the Virtual Machine')
@secure()
param adminPasswordOrKey string

@description('The size of the VM')
param vmSize string = 'Standard_B1s'

@description('The container image reference to deploy (e.g., ghcr.io/<owner>/<repo>:latest)')
param dockerImage string

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

@description('Whether to create and attach a Public IP address (defaults to false)')
param enablePublicIP bool = false

var publicIPAddressName = '${vmName}PublicIP'
var networkInterfaceName = '${vmName}NetInt'
var osDiskType = 'Standard_LRS'
var virtualNetworkName = '${vmName}VNet'
var subnetName = 'Subnet'
var networkSecurityGroupName = '${vmName}NSG'

// 1. Networking
resource publicIP 'Microsoft.Network/publicIPAddresses@2024-03-01' = if (enablePublicIP) {
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
          publicIPAddress: enablePublicIP ? {
            id: publicIP.id
          } : null
        }
      }
    ]
  }
}

// 2. Virtual Machine
resource vm 'Microsoft.Compute/virtualMachines@2024-03-01' = {
  name: vmName
  location: location
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

// 3. Custom Script Extension (Encrypted In-Transit via protectedSettings)
var rawScript = '''#!/bin/bash
set -e

# Step 1: Install Docker and dependencies
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
}

# Step 2: Write all Configuration files and in-memory secrets
write_configs() {
    mkdir -p /hurion/app

    cat <<'EOF' > /hurion/app/compose.yaml
{{composeProdYaml}}
EOF

    cat <<'EOF' > /hurion/app/config.yaml
{{configYaml}}
EOF

    cat <<'EOF' > /hurion/app/.env
{{appEnvContent}}
EOF

    echo "" >> /hurion/app/.env
    echo "IMAGE_NAME={{dockerImage}}" >> /hurion/app/.env

    if [ -n "{{dockerEnvContent}}" ]; then
        cat <<'EOF' > /hurion/app/.docker.env
{{dockerEnvContent}}
EOF
    else
        rm -f /hurion/app/.docker.env
    fi
}

# Step 3: Login to private Docker Hub registry if credentials exist
docker_login() {
    if [ -f /hurion/app/.docker.env ]; then
        local user=$(grep -E "^DOCKER_USER=" /hurion/app/.docker.env | head -n 1 | cut -d"=" -f2- | sed -e 's/[#].*$//' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' | tr -d '\r\n')
        local pat=$(grep -E "^DOCKER_PAT=" /hurion/app/.docker.env | head -n 1 | cut -d"=" -f2- | sed -e 's/[#].*$//' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' | tr -d '\r\n')
        local server=$(grep -E "^DOCKER_SERVER=" /hurion/app/.docker.env | head -n 1 | cut -d"=" -f2- | sed -e 's/[#].*$//' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' | tr -d '\r\n')
        
        if [ -n "$user" ] && [ -n "$pat" ]; then
            if [ -n "$server" ]; then
                echo "$pat" | docker login "$server" -u "$user" --password-stdin
            else
                echo "$pat" | docker login -u "$user" --password-stdin
            fi
        fi
    fi
}

# Step 4: Start / Restart the containerized bot
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
docker_login
run_application
'''

var script1 = replace(rawScript, '{{adminUsername}}', adminUsername)
var script2 = replace(script1, '{{composeProdYaml}}', composeProdYaml)
var script3 = replace(script2, '{{configYaml}}', configYaml)
var script4 = replace(script3, '{{appEnvContent}}', appEnvContent)
var script5 = replace(script4, '{{dockerEnvContent}}', dockerEnvContent)
var finalScript = replace(script5, '{{dockerImage}}', dockerImage)

resource updateScript 'Microsoft.Compute/virtualMachines/extensions@2024-03-01' = {
  parent: vm
  name: 'UpdateHurion'
  location: location
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

output publicIP string = enablePublicIP ? publicIP!.properties.ipAddress : 'none'
