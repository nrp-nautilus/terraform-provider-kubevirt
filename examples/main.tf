terraform {
  required_providers {
    kubevirt = {
      source  = "gitlab.nrp-nautilus.io/nrp/kubevirt"
    }
  }
}

provider "kubevirt" {
  namespace = "terraform-dev"
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
