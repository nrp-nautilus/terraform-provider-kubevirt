package kubevirt

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/nrp-nautilus/terraform-provider-kubevirt/kubevirt/client"
	"github.com/nrp-nautilus/terraform-provider-kubevirt/kubevirt/schema/virtualmachine"
	"github.com/nrp-nautilus/terraform-provider-kubevirt/kubevirt/utils"
	"k8s.io/apimachinery/pkg/api/errors"
)

func resourceKubevirtVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubevirtVirtualMachineCreate,
		Read:   resourceKubevirtVirtualMachineRead,
		Update: resourceKubevirtVirtualMachineUpdate,
		Delete: resourceKubevirtVirtualMachineDelete,
		Exists: resourceKubevirtVirtualMachineExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Schema: virtualmachine.VirtualMachineFields(),
	}
}

func resourceKubevirtVirtualMachineCreate(resourceData *schema.ResourceData, meta interface{}) error {
	cli := (meta).(client.Client)

	vm, err := virtualmachine.FromResourceData(resourceData)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating new virtual machine: %#v", vm)
	if err := cli.CreateVirtualMachine(vm); err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new virtual machine: %#v", vm)
	if err := virtualmachine.ToResourceData(*vm, resourceData); err != nil {
		return err
	}
	resourceData.SetId(utils.BuildId(vm.ObjectMeta))

	log.Printf("[INFO] Successfully created virtual machine: %s", vm.Name)
	return resourceKubevirtVirtualMachineRead(resourceData, meta)
}

func resourceKubevirtVirtualMachineRead(resourceData *schema.ResourceData, meta interface{}) error {
	cli := (meta).(client.Client)

	namespace, name, err := utils.IdParts(resourceData.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading virtual machine %s", name)

	vm, err := cli.GetVirtualMachine(namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[WARN] Virtual machine %s not found, removing from state", name)
			resourceData.SetId("")
			return nil
		}
		return fmt.Errorf("failed to read virtual machine: %v", err)
	}

	if err := virtualmachine.ToResourceData(*vm, resourceData); err != nil {
		return fmt.Errorf("failed to convert virtual machine to resource data: %v", err)
	}

	return nil
}

func resourceKubevirtVirtualMachineUpdate(resourceData *schema.ResourceData, meta interface{}) error {
	cli := (meta).(client.Client)

	namespace, name, err := utils.IdParts(resourceData.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Updating virtual machine %s", name)

	// Get current VM to get the resource version
	currentVM, err := cli.GetVirtualMachine(namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get current virtual machine: %v", err)
	}

	// Create updated VM object
	updatedVM, err := virtualmachine.FromResourceData(resourceData)
	if err != nil {
		return fmt.Errorf("failed to create updated virtual machine object: %v", err)
	}

	// Preserve the resource version for update
	updatedVM.ObjectMeta.ResourceVersion = currentVM.ObjectMeta.ResourceVersion

	// Update the VM
	if err := cli.UpdateVirtualMachine(namespace, name, updatedVM, nil); err != nil {
		return fmt.Errorf("failed to update virtual machine: %v", err)
	}

	log.Printf("[INFO] Successfully updated virtual machine: %s", name)
	return resourceKubevirtVirtualMachineRead(resourceData, meta)
}

func resourceKubevirtVirtualMachineDelete(resourceData *schema.ResourceData, meta interface{}) error {
	cli := (meta).(client.Client)

	namespace, name, err := utils.IdParts(resourceData.Id())
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting virtual machine %s", name)

	if err := cli.DeleteVirtualMachine(namespace, name); err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[WARN] Virtual machine %s not found during deletion", name)
			return nil
		}
		return fmt.Errorf("failed to delete virtual machine: %v", err)
	}

	log.Printf("[INFO] Successfully deleted virtual machine: %s", name)
	return nil
}

func resourceKubevirtVirtualMachineExists(resourceData *schema.ResourceData, meta interface{}) (bool, error) {
	cli := (meta).(client.Client)

	namespace, name, err := utils.IdParts(resourceData.Id())
	if err != nil {
		return false, err
	}

	_, err = cli.GetVirtualMachine(namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check virtual machine existence: %v", err)
	}

	return true, nil
}
