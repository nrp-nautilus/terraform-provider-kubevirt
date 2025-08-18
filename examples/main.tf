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

# Test the custom provider by creating a Kubernetes pod
resource "kubevirt_kubernetes_pod" "test" {
  name      = "test-pod"
  namespace = "terraform-dev"
  image     = "busybox:latest"
  command   = ["/bin/sh"]
  args      = ["-c", "echo 'Hello from custom provider pod!' && sleep 3600"]
}

# Output the result
output "pod_name" {
  value = kubevirt_kubernetes_pod.test.name
}

output "pod_status" {
  value = kubevirt_kubernetes_pod.test.pod_status
}
