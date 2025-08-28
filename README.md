# KubeVirt Terraform Provider

A minimal Terraform provider example that demonstrates how to create custom providers for use with Coder.

## Features

- **Hello World Resource**: A simple resource that stores a configurable message
- **Hello World Data Source**: A data source that returns example data
- **GitLab CI/CD**: Automated build and release pipeline

## Quick Start

### 1. Build the Provider

```bash
# Install Go 1.21+
go mod download
go build -o terraform-provider-kubevirt .
```

### 2. Test Locally

```bash
# Create a test directory
mkdir test-provider && cd test-provider

# Copy the provider binary
cp ../terraform-provider-kubevirt .

# Create a simple test configuration
cat > main.tf <<EOF
terraform {
  required_providers {
    kubevirt = {
      source = "./terraform-provider-kubevirt"
    }
  }
}

resource "kubevirt_hello_world" "test" {
  configurable_attribute = "Hello World!"
}
EOF

# Initialize and apply
terraform init
terraform plan
terraform apply
```

### 3. Use in Coder

To use this provider in your Coder templates:

```hcl
terraform {
  required_providers {
    kubevirt = {
      source  = "gitlab.nrp-nautilus.io/terraform-dev/kubevirt-terraform-provider/kubevirt"
      version = "0.1.0"
    }
  }
}

provider "kubevirt" {
  namespace = "terraform-dev"
}

resource "kubevirt_hello_world" "example" {
  configurable_attribute = "Hello from Coder!"
}
```

## GitLab CI/CD Pipeline

The pipeline automatically:

1. **Tests**: Runs `go test`, `go vet`, and `go fmt`
2. **Builds**: Creates binaries for Linux and macOS
3. **Releases**: Creates GitLab releases when you tag commits

### To Release a New Version

```bash
git tag v0.1.0
git push origin v0.1.0
```

The pipeline will automatically build and create a release with the provider binaries.

## Provider Structure

```
.
├── main.go                    # Provider entry point
├── provider/
│   ├── provider.go           # Main provider implementation
│   ├── resource_hello_world.go # Hello World resource
│   └── data_source_hello_world.go # Hello World data source
├── examples/
│   └── main.tf              # Example usage
├── go.mod                    # Go dependencies
├── .gitlab-ci.yml           # CI/CD pipeline
└── README.md                # This file
```

## Customization

To add your own resources:

1. Create a new file in the `provider/` directory (e.g., `resource_virtualmachine.go`)
2. Implement the resource interface
3. Add it to the provider's `Resources()` method
4. Update the provider schema if needed

## Next Steps

This is a minimal example. To create a full KubeVirt provider:

1. Add Kubernetes client configuration
2. Implement VirtualMachine resource with proper CRUD operations
3. Add support for sidecar hooks and PCI device passthrough
4. Implement proper error handling and validation
5. Add comprehensive testing

## Troubleshooting

### Provider Not Found

Make sure the provider binary is in the correct location and has the right name format:
`terraform-provider-{name}_{version}`

### Build Errors

Ensure you have Go 1.21+ installed and all dependencies are downloaded:
```bash
go mod download
go mod verify
```

### CI/CD Pipeline Issues

Check that:
- Your GitLab project has the necessary permissions
- The `CI_JOB_TOKEN` has access to create releases
- The Go version in the pipeline matches your local version
# Trigger new workflow run
# Force new workflow run - Thu Aug 21 23:54:14 EDT 2025
# Trigger new build Wed Aug 27 17:50:53 EDT 2025
# Trigger build with manifest fix Wed Aug 27 18:01:12 EDT 2025
# Test current workflow Wed Aug 27 19:08:00 EDT 2025
# Test clean workflow Wed Aug 27 19:11:08 EDT 2025
# Test new clean workflow Wed Aug 27 19:15:28 EDT 2025
# Test fixed workflow with binary renaming Wed Aug 27 20:57:44 EDT 2025
# Test workflow with manually inserted binary renaming Wed Aug 27 21:02:57 EDT 2025
# Test workflow with correct step order Wed Aug 27 21:10:00 EDT 2025
# Force workflow refresh Wed Aug 27 21:15:59 EDT 2025
# Test completely new workflow with correct step order Wed Aug 27 21:24:44 EDT 2025
# Test workflow with unzip fix Wed Aug 27 21:36:32 EDT 2025
# Test workflow with unzip -o fix Wed Aug 27 21:39:00 EDT 2025
# Test workflow with find fix for binary names Wed Aug 27 21:41:38 EDT 2025
# Fixed GPG signing template issue
Force updating .goreleaser.yml
