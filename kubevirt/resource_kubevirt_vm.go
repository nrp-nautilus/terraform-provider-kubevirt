package kubevirt

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/nrp-nautilus/terraform-provider-kubevirt/kubevirt/client"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func resourceKubevirtKubevirtVM() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubevirtKubevirtVMCreate,
		Read:   resourceKubevirtKubevirtVMRead,
		Update: resourceKubevirtKubevirtVMUpdate,
		Delete: resourceKubevirtKubevirtVMDelete,
		Exists: resourceKubevirtKubevirtVMExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier for the VM",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the VirtualMachine",
			},
			"namespace": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Kubernetes namespace for the VM",
			},
			"image": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Container image for the VM",
			},
			"memory": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Memory allocation for the VM (e.g., '1Gi', '512Mi')",
			},
			"cpu": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Number of CPU cores for the VM",
			},
			"machine_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "q35",
				Description: "Machine type for the VM (e.g., 'q35', 'pc-q35-rhel8.0')",
			},
			"architecture": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "amd64",
				Description: "CPU architecture for the VM",
			},
			"hugepages": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Hugepages configuration (e.g., '2Mi', '1Gi')",
			},
			"sidecar_hook": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Sidecar hook script name (ConfigMap)",
			},
			"node_selector": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Node selector for VM placement",
			},
			"tolerations": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Tolerations for node scheduling",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Toleration key",
						},
						"operator": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "Equal",
							Description: "Toleration operator (Equal, Exists)",
						},
						"value": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Toleration value",
						},
						"effect": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Toleration effect (NoSchedule, PreferNoSchedule, NoExecute)",
						},
					},
				},
			},
			"affinity": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Affinity for node and pod scheduling",
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"node_affinity": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Node affinity scheduling rules",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"required_during_scheduling_ignored_during_execution": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Required node affinity rules",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"node_selector_terms": {
													Type:        schema.TypeList,
													Required:    true,
													Description: "Node selector terms",
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"match_expressions": {
																Type:        schema.TypeList,
																Optional:    true,
																Description: "Node selector expressions",
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"key": {
																			Type:        schema.TypeString,
																			Required:    true,
																			Description: "Label key",
																		},
																		"operator": {
																			Type:        schema.TypeString,
																			Required:    true,
																			Description: "Operator (In, NotIn, Exists, DoesNotExist, Gt, Lt)",
																		},
																		"values": {
																			Type:        schema.TypeList,
																			Optional:    true,
																			Description: "Label values",
																			Elem:        &schema.Schema{Type: schema.TypeString},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"pod_affinity": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Pod affinity scheduling rules",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"required_during_scheduling_ignored_during_execution": {
										Type:        schema.TypeList,
										Optional:    true,
										Description: "Required pod affinity rules",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"label_selector": {
													Type:        schema.TypeList,
													Optional:    true,
													Description: "Label selector",
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"match_expressions": {
																Type:        schema.TypeList,
																Optional:    true,
																Description: "Match expressions",
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"key": {
																			Type:        schema.TypeString,
																			Required:    true,
																			Description: "Label key",
																		},
																		"operator": {
																			Type:        schema.TypeString,
																			Required:    true,
																			Description: "Operator (In, NotIn, Exists, DoesNotExist)",
																		},
																		"values": {
																			Type:        schema.TypeList,
																			Optional:    true,
																			Description: "Label values",
																			Elem:        &schema.Schema{Type: schema.TypeString},
																		},
																	},
																},
															},
														},
													},
												},
												"namespaces": {
													Type:        schema.TypeList,
													Optional:    true,
													Description: "Namespaces to match",
													Elem:        &schema.Schema{Type: schema.TypeString},
												},
												"topology_key": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "Topology key for affinity",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"host_devices": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Host devices to attach to the VM",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the host device",
						},
						"device_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Device name on the host",
						},
					},
				},
			},
			"usb_devices": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "USB devices to attach to the VM",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vendor_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "USB vendor ID",
						},
						"product_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "USB product ID",
						},
					},
				},
			},
			"pci_devices": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "PCI devices to attach to the VM",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the PCI device",
						},
						"device_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Device name on the host",
						},
						"vendor_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "PCI vendor ID (hex format, e.g., '10de' for NVIDIA)",
						},
						"product_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "PCI product ID (hex format)",
						},
					},
				},
			},
			"gpu_devices": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "GPU devices to attach to the VM",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the GPU device",
						},
						"device_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Device name on the host",
						},
						"vendor_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "GPU vendor ID (e.g., '10de' for NVIDIA, '1002' for AMD)",
						},
						"product_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "GPU product ID",
						},
					},
				},
			},
			"network_interfaces": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Network interfaces for the VM",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the network interface",
						},
						"network_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Network name to attach to",
						},
					},
				},
			},
			"cloud_init": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Cloud-init configuration for the VM",
			},
			"coder_agent_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Coder agent token for workspace integration",
			},
			"vm_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current status of the VM",
			},
			"creation_timestamp": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when the VM was created",
			},
			"workspace_transition": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current workspace transition state",
			},
		},
	}
}

func resourceKubevirtKubevirtVMCreate(d *schema.ResourceData, meta interface{}) error {
	cli := meta.(client.Client)
	
	name := d.Get("name").(string)
	namespace := d.Get("namespace").(string)
	
	log.Printf("[INFO] Creating KubeVirt VM: %s in namespace: %s", name, namespace)
	
	// Create the VM object
	vm, err := createVMObject(d)
	if err != nil {
		return fmt.Errorf("failed to create VM object: %v", err)
	}
	
	// Create the VM in Kubernetes
	dynamicClient, err := getDynamicClient(cli)
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %v", err)
	}
	
	vmResource := dynamicClient.Resource(k8sschema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	})
	
	_, err = vmResource.Namespace(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create VM: %v", err)
	}
	
	// Set the ID
	d.SetId(fmt.Sprintf("%s/%s", namespace, name))
	
	log.Printf("[INFO] Successfully created KubeVirt VM: %s", name)
	
	return resourceKubevirtKubevirtVMRead(d, meta)
}

func resourceKubevirtKubevirtVMRead(d *schema.ResourceData, meta interface{}) error {
	cli := meta.(client.Client)
	
	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	
	log.Printf("[INFO] Reading KubeVirt VM: %s in namespace: %s", name, namespace)
	
	dynamicClient, err := getDynamicClient(cli)
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %v", err)
	}
	
	vmResource := dynamicClient.Resource(k8sschema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	})
	
	vm, err := vmResource.Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[WARN] KubeVirt VM %s not found, removing from state", name)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("failed to read VM: %v", err)
	}
	
	// Update the resource data
	if err := updateResourceDataFromVM(vm, d); err != nil {
		return fmt.Errorf("failed to update resource data: %v", err)
	}
	
	return nil
}

func resourceKubevirtKubevirtVMUpdate(d *schema.ResourceData, meta interface{}) error {
	cli := meta.(client.Client)
	
	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	
	log.Printf("[INFO] Updating KubeVirt VM: %s in namespace: %s", name, namespace)
	
	// Get the current VM
	dynamicClient, err := getDynamicClient(cli)
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %v", err)
	}
	
	vmResource := dynamicClient.Resource(k8sschema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	})
	
	_, err = vmResource.Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get current VM: %v", err)
	}
	
	// Create updated VM object
	updatedVM, err := createVMObject(d)
	if err != nil {
		return fmt.Errorf("failed to create updated VM object: %v", err)
	}
	
	// Update the VM
	_, err = vmResource.Namespace(namespace).Update(context.Background(), updatedVM, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update VM: %v", err)
	}
	
	log.Printf("[INFO] Successfully updated KubeVirt VM: %s", name)
	
	return resourceKubevirtKubevirtVMRead(d, meta)
}

func resourceKubevirtKubevirtVMDelete(d *schema.ResourceData, meta interface{}) error {
	cli := meta.(client.Client)
	
	namespace, name, err := idParts(d.Id())
	if err != nil {
		return err
	}
	
	log.Printf("[INFO] Deleting KubeVirt VM: %s in namespace: %s", name, namespace)
	
	dynamicClient, err := getDynamicClient(cli)
	if err != nil {
		return fmt.Errorf("failed to get dynamic client: %v", err)
	}
	
	vmResource := dynamicClient.Resource(k8sschema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	})
	
	err = vmResource.Namespace(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("[WARN] KubeVirt VM %s not found during deletion", name)
			return nil
		}
		return fmt.Errorf("failed to delete VM: %v", err)
	}
	
	log.Printf("[INFO] Successfully deleted KubeVirt VM: %s", name)
	
	return nil
}

func resourceKubevirtKubevirtVMExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	cli := meta.(client.Client)
	
	namespace, name, err := idParts(d.Id())
	if err != nil {
		return false, err
	}
	
	dynamicClient, err := getDynamicClient(cli)
	if err != nil {
		return false, fmt.Errorf("failed to get dynamic client: %v", err)
	}
	
	vmResource := dynamicClient.Resource(k8sschema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	})
	
	_, err = vmResource.Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check VM existence: %v", err)
	}
	
	return true, nil
}

// Helper functions
func createVMObject(d *schema.ResourceData) (*unstructured.Unstructured, error) {
	vm := &unstructured.Unstructured{}
	vm.SetAPIVersion("kubevirt.io/v1")
	vm.SetKind("VirtualMachine")
	vm.SetName(d.Get("name").(string))
	vm.SetNamespace(d.Get("namespace").(string))
	
	// Set labels
	vm.SetLabels(map[string]string{
		"app": "kubevirt-vm",
		"managed-by": "terraform",
	})
	
	// Create spec
	spec := map[string]interface{}{
		"running": false,
		"template": map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": map[string]interface{}{
					"kubevirt.io/vm": d.Get("name").(string),
				},
			},
			"spec": map[string]interface{}{
				"domain": map[string]interface{}{
					"devices": map[string]interface{}{
						"disks": []map[string]interface{}{
							{
								"name": "containerdisk",
								"disk": map[string]interface{}{},
							},
						},
					},
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"memory": d.Get("memory").(string),
							"cpu":    d.Get("cpu").(int),
						},
					},
				},
				"volumes": []map[string]interface{}{
					{
						"name": "containerdisk",
						"containerDisk": map[string]interface{}{
							"image": d.Get("image").(string),
						},
					},
				},
			},
		},
	}
	
	// Add sidecar hook if specified
	if sidecarHook, ok := d.GetOk("sidecar_hook"); ok && sidecarHook.(string) != "" {
		annotations := map[string]string{
			"hooks.kubevirt.io/hookSidecars": fmt.Sprintf(`[{"args":["--version","v1alpha2"],"configMap":{"hookPath":"/usr/bin/onDefineDomain","key":"%s.py","name":"%s"}}]`, sidecarHook.(string), sidecarHook.(string)),
		}
		spec["template"].(map[string]interface{})["metadata"].(map[string]interface{})["annotations"] = annotations
	}
	
	// Add tolerations if specified
	if tolerations, ok := d.GetOk("tolerations"); ok && len(tolerations.([]interface{})) > 0 {
		var tolObjects []map[string]interface{}
		for _, tol := range tolerations.([]interface{}) {
			tolMap := tol.(map[string]interface{})
			tolObject := map[string]interface{}{
				"operator": "Exists", // Default to Exists for key-only tolerations
			}
			
			if key, ok := tolMap["key"].(string); ok && key != "" {
				tolObject["key"] = key
			}
			if operator, ok := tolMap["operator"].(string); ok && operator != "" {
				tolObject["operator"] = operator
			}
			if value, ok := tolMap["value"].(string); ok && value != "" {
				tolObject["value"] = value
			}
			if effect, ok := tolMap["effect"].(string); ok && effect != "" {
				tolObject["effect"] = effect
			}
			
			tolObjects = append(tolObjects, tolObject)
		}
		
		if len(tolObjects) > 0 {
			spec["template"].(map[string]interface{})["spec"].(map[string]interface{})["tolerations"] = tolObjects
		}
	}
	
	// Add node selector if specified
	if nodeSelector, ok := d.GetOk("node_selector"); ok && len(nodeSelector.(map[string]interface{})) > 0 {
		spec["template"].(map[string]interface{})["spec"].(map[string]interface{})["nodeSelector"] = nodeSelector.(map[string]interface{})
	}
	
	// Add affinity if specified
	if affinity, ok := d.GetOk("affinity"); ok && len(affinity.([]interface{})) > 0 {
		if affinityList := affinity.([]interface{}); len(affinityList) > 0 && affinityList[0] != nil {
			affinityMap := affinityList[0].(map[string]interface{})
			spec["template"].(map[string]interface{})["spec"].(map[string]interface{})["affinity"] = affinityMap
		}
	}
	
	// Add PCI devices if specified
	if pciDevices, ok := d.GetOk("pci_devices"); ok && len(pciDevices.([]interface{})) > 0 {
		var pciObjects []map[string]interface{}
		for _, pci := range pciDevices.([]interface{}) {
			pciMap := pci.(map[string]interface{})
			pciObject := map[string]interface{}{
				"name":        pciMap["name"].(string),
				"deviceName":  pciMap["device_name"].(string),
			}
			
			if vendorID, ok := pciMap["vendor_id"].(string); ok && vendorID != "" {
				pciObject["vendorId"] = vendorID
			}
			if productID, ok := pciMap["product_id"].(string); ok && productID != "" {
				pciObject["productId"] = productID
			}
			
			pciObjects = append(pciObjects, pciObject)
		}
		
		if len(pciObjects) > 0 {
			spec["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["devices"].(map[string]interface{})["hostDevices"] = pciObjects
		}
	}
	
	// Add GPU devices if specified
	if gpuDevices, ok := d.GetOk("gpu_devices"); ok && len(gpuDevices.([]interface{})) > 0 {
		var gpuObjects []map[string]interface{}
		for _, gpu := range gpuDevices.([]interface{}) {
			gpuMap := gpu.(map[string]interface{})
			gpuObject := map[string]interface{}{
				"name":        gpuMap["name"].(string),
				"deviceName":  gpuMap["device_name"].(string),
			}
			
			if vendorID, ok := gpuMap["vendor_id"].(string); ok && vendorID != "" {
				gpuObject["vendorId"] = vendorID
			}
			if productID, ok := gpuMap["product_id"].(string); ok && productID != "" {
				gpuObject["productId"] = productID
			}
			
			gpuObjects = append(gpuObjects, gpuObject)
		}
		
		if len(gpuObjects) > 0 {
			// Add to existing hostDevices or create new
			if existingHostDevices, ok := spec["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["devices"].(map[string]interface{})["hostDevices"]; ok {
				if existingList, ok := existingHostDevices.([]map[string]interface{}); ok {
					existingList = append(existingList, gpuObjects...)
					spec["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["devices"].(map[string]interface{})["hostDevices"] = existingList
				}
			} else {
				spec["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["devices"].(map[string]interface{})["hostDevices"] = gpuObjects
			}
		}
	}
	
	vm.Object["spec"] = spec
	
	return vm, nil
}

func updateResourceDataFromVM(vm *unstructured.Unstructured, d *schema.ResourceData) error {
	// Extract basic fields
	if err := d.Set("name", vm.GetName()); err != nil {
		return err
	}
	if err := d.Set("namespace", vm.GetNamespace()); err != nil {
		return err
	}
	if err := d.Set("creation_timestamp", vm.GetCreationTimestamp().String()); err != nil {
		return err
	}
	
	// Extract spec fields
	if spec, ok := vm.Object["spec"].(map[string]interface{}); ok {
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if templateSpec, ok := template["spec"].(map[string]interface{}); ok {
				if domain, ok := templateSpec["domain"].(map[string]interface{}); ok {
					if resources, ok := domain["resources"].(map[string]interface{}); ok {
						if requests, ok := resources["requests"].(map[string]interface{}); ok {
							if memory, ok := requests["memory"].(string); ok {
								if err := d.Set("memory", memory); err != nil {
									return err
								}
							}
							if cpu, ok := requests["cpu"].(string); ok && cpu != "" {
								if cpuInt, err := strconv.Atoi(cpu); err == nil {
									if err := d.Set("cpu", cpuInt); err != nil {
										return err
									}
								} else {
									log.Printf("[WARN] Failed to convert CPU value '%s' to int: %v", cpu, err)
								}
							} else {
								log.Printf("[DEBUG] CPU value not found or empty in requests")
							}
						}
					}
				}
			}
		}
	}
	
	// Extract volumes for image
	if spec, ok := vm.Object["spec"].(map[string]interface{}); ok {
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if templateSpec, ok := template["spec"].(map[string]interface{}); ok {
				if volumes, ok := templateSpec["volumes"].([]interface{}); ok {
					for _, volume := range volumes {
						if volMap, ok := volume.(map[string]interface{}); ok {
							if containerDisk, ok := volMap["containerDisk"].(map[string]interface{}); ok {
								if image, ok := containerDisk["image"].(string); ok {
									if err := d.Set("image", image); err != nil {
										return err
									}
									break
								}
							}
						}
					}
				}
			}
		}
	}
	
	return nil
}

func getDynamicClient(cli client.Client) (dynamic.Interface, error) {
	return cli.GetDynamicClient(), nil
}

func idParts(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ID format: %s, expected namespace/name", id)
	}
	return parts[0], parts[1], nil
}
