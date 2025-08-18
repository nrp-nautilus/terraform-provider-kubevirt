terraform {
  required_providers {
    kubevirt = {
      source  = "gitlab.nrp-nautilus.io/nrp/kubevirt-terraform-provider/kubevirt"
      version = "0.1.0"
    }
  }
}

provider "kubevirt" {
  # Optional: Add any provider configuration here
}

# Example resource
resource "kubevirt_hello_world" "example" {
  configurable_attribute = "Hello from KubeVirt Provider!"
}

# Example data source
data "kubevirt_hello_world" "example" {
  configurable_attribute = "Hello from Data Source!"
}

output "resource_id" {
  value = kubevirt_hello_world.example.id
}

output "data_source_id" {
  value = data.kubevirt_hello_world.example.id
}
