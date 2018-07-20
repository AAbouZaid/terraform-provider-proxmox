package proxmox

import (
	"fmt"
	pxapi "github.com/Telmate/proxmox-api-go/proxmox"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"path"
	"strconv"
	"strings"
	"time"
)

func resourceVmQemu() *schema.Resource {
	*pxapi.Debug = true
	return &schema.Resource{
		Create: resourceVmQemuCreate,
		Read:   resourceVmQemuRead,
		Update: resourceVmQemuUpdate,
		Delete: resourceVmQemuDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVmQemuImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"desc": {
				Type:     schema.TypeString,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"target_node": {
				Type:     schema.TypeString,
				Required: true,
			},
			"ssh_forward_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"onboot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"iso": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"clone": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"storage": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"qemu_os": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "l26",
			},
			"memory": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"cores": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"sockets": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"disk_gb": {
				Type:     schema.TypeFloat,
				Optional: true,
			},
			"nic": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use `network` instead",
			},
			"bridge": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use `network.bridge` instead",
			},
			"vlan": {
				Type:       schema.TypeInt,
				Optional:   true,
				Default:    -1,
				Deprecated: "Use `network.tag` instead",
			},
			"network": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"bridge": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "nat",
						},
						"tag": &schema.Schema{
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "VLAN tag.",
							Default:     -1,
						},
						"firewall": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
						"rate": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
						"queues": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
						"link_down": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
					},
				},
			},
			"disk": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"storage": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"size": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"cache": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "none",
						},
						"backup": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
						"iothread": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
						"replicate": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  -1,
						},
					},
				},
			},
			"os_type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"os_network_config": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"ssh_user": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"ssh_private_key": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.TrimSpace(old) == strings.TrimSpace(new)
				},
			},
			"force_create": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"preprovision": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceVmQemuCreate(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	pmParallelBegin(pconf)
	client := pconf.Client
	vmName := d.Get("name").(string)
	disk_gb := d.Get("disk_gb").(float64)

	networks := d.Get("network").(*schema.Set)
	qemuNetworks := devicesSetToMap(networks)
	disks := d.Get("disk").(*schema.Set)
	qemuDisks := devicesSetToMap(disks)

	config := pxapi.ConfigQemu{
		Name:         vmName,
		Description:  d.Get("desc").(string),
		Onboot:       Btoi(d.Get("onboot").(bool)),
		Storage:      d.Get("storage").(string),
		Memory:       d.Get("memory").(int),
		QemuCores:    d.Get("cores").(int),
		QemuSockets:  d.Get("sockets").(int),
		QemuOs:       d.Get("qemu_os").(string),
		QemuNetworks: qemuNetworks,
		QemuDisks:    qemuDisks,
		// Deprecated.
		QemuNicModel: d.Get("nic").(string),
		QemuBrige:    d.Get("bridge").(string),
		QemuVlanTag:  d.Get("vlan").(int),
		DiskSize:     disk_gb,
	}
	log.Print("[DEBUG] checking for duplicate name")
	dupVmr, _ := client.GetVmRefByName(vmName)

	forceCreate := d.Get("force_create").(bool)
	targetNode := d.Get("target_node").(string)

	if dupVmr != nil && forceCreate {
		pmParallelEnd(pconf)
		return fmt.Errorf("Duplicate VM name (%s) with vmId: %d. Set force_create=false to recycle", vmName, dupVmr.VmId())
	} else if dupVmr != nil && dupVmr.Node() != targetNode {
		pmParallelEnd(pconf)
		return fmt.Errorf("Duplicate VM name (%s) with vmId: %d on different target_node=%s", vmName, dupVmr.VmId(), dupVmr.Node())
	}

	vmr := dupVmr

	if vmr == nil {
		// get unique id
		nextid, err := nextVmId(pconf)
		if err != nil {
			pmParallelEnd(pconf)
			return err
		}
		vmr = pxapi.NewVmRef(nextid)

		vmr.SetNode(targetNode)
		// check if ISO or clone
		if d.Get("clone").(string) != "" {
			sourceVmr, err := client.GetVmRefByName(d.Get("clone").(string))
			if err != nil {
				pmParallelEnd(pconf)
				return err
			}
			log.Print("[DEBUG] cloning VM")
			err = config.CloneVm(sourceVmr, vmr, client)
			if err != nil {
				pmParallelEnd(pconf)
				return err
			}

			// give sometime to proxmox to catchup
			time.Sleep(5 * time.Second)

			err = prepareDiskSize(client, vmr, disk_gb)
			if err != nil {
				pmParallelEnd(pconf)
				return err
			}

		} else if d.Get("iso").(string) != "" {
			config.QemuIso = d.Get("iso").(string)
			err := config.CreateVm(vmr, client)
			if err != nil {
				pmParallelEnd(pconf)
				return err
			}
		}
	} else {
		log.Printf("[DEBUG] recycling VM vmId: %d", vmr.VmId())

		client.StopVm(vmr)

		err := config.UpdateConfig(vmr, client)
		if err != nil {
			pmParallelEnd(pconf)
			return err
		}

		// give sometime to proxmox to catchup
		time.Sleep(5 * time.Second)

		err = prepareDiskSize(client, vmr, disk_gb)
		if err != nil {
			pmParallelEnd(pconf)
			return err
		}
	}
	d.SetId(resourceId(targetNode, "qemu", vmr.VmId()))

	// give sometime to proxmox to catchup
	time.Sleep(5 * time.Second)

	log.Print("[DEBUG] starting VM")
	_, err := client.StartVm(vmr)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	preProvision := d.Get("preprovision").(bool)
	if preProvision {
		log.Print("[DEBUG] setting up SSH forward")
		sshPort, err := pxapi.SshForwardUsernet(vmr, client)
		if err != nil {
			pmParallelEnd(pconf)
			return err
		}

		// Done with proxmox API, end parallel and do the SSH things
		pmParallelEnd(pconf)

		d.SetConnInfo(map[string]string{
			"type":        "ssh",
			"host":        d.Get("ssh_forward_ip").(string),
			"port":        sshPort,
			"user":        d.Get("ssh_user").(string),
			"private_key": d.Get("ssh_private_key").(string),
			"pm_api_url":  client.ApiUrl,
			"pm_user":     client.Username,
			"pm_password": client.Password,
		})

		switch d.Get("os_type").(string) {

		case "ubuntu":
			// give sometime to bootup
			time.Sleep(9 * time.Second)
			err = preProvisionUbuntu(d)
			if err != nil {
				return err
			}

		case "centos":
			// give sometime to bootup
			time.Sleep(9 * time.Second)
			err = preProvisionCentos(d)
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("Unknown os_type: %s", d.Get("os_type").(string))
		}
	}
	return nil
}

func resourceVmQemuUpdate(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	pmParallelBegin(pconf)
	client := pconf.Client
	vmr, err := client.GetVmRefByName(d.Get("name").(string))
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	vmName := d.Get("name").(string)
	disk_gb := d.Get("disk_gb").(float64)

	//activeConf, err := pxapi.NewConfigQemuFromApi(vmr, client)

	configDisksSet := d.Get("disk").(*schema.Set)
	configDisksMap := devicesSetToMap(configDisksSet)

	configNetworksSet := d.Get("network").(*schema.Set)
	configNetworksMap := devicesSetToMap(configNetworksSet)

	//
	// TODO: Retain mac adresses.
	//currentConf, err := pxapi.NewConfigQemuFromApi(vmr, client)
	//macAddrss := map[int]string{}
	//for nicID, nicConfMap := range currentConf.QemuNetworks {
	//	nicType := nicConfMap["type"].(string)
	//	typeConf := strings.Split(nicType, "=")
	//	macAddrss[nicID] = typeConf[1]
	//}

	config := pxapi.ConfigQemu{
		Name:         vmName,
		Description:  d.Get("desc").(string),
		Onboot:       Btoi(d.Get("onboot").(bool)),
		Storage:      d.Get("storage").(string),
		Memory:       d.Get("memory").(int),
		QemuCores:    d.Get("cores").(int),
		QemuSockets:  d.Get("sockets").(int),
		QemuOs:       d.Get("qemu_os").(string),
		QemuDisks:    configDisksMap,
		QemuNetworks: configNetworksMap,
		// Deprecated.
		QemuNicModel: d.Get("nic").(string),
		QemuBrige:    d.Get("bridge").(string),
		QemuVlanTag:  d.Get("vlan").(int),
		DiskSize:     disk_gb,
	}

	err = config.UpdateConfig(vmr, client)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	// give sometime to proxmox to catchup
	time.Sleep(5 * time.Second)

	prepareDiskSize(client, vmr, disk_gb)

	// give sometime to proxmox to catchup
	time.Sleep(5 * time.Second)

	log.Print("[DEBUG] starting VM")
	_, err = client.StartVm(vmr)

	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	preProvision := d.Get("preprovision").(bool)
	if preProvision {
		sshPort, err := pxapi.SshForwardUsernet(vmr, client)
		if err != nil {
			pmParallelEnd(pconf)
			return err
		}
		d.SetConnInfo(map[string]string{
			"type":        "ssh",
			"host":        d.Get("ssh_forward_ip").(string),
			"port":        sshPort,
			"user":        d.Get("ssh_user").(string),
			"private_key": d.Get("ssh_private_key").(string),
		})

		pmParallelEnd(pconf)
	}

	// give sometime to bootup
	time.Sleep(9 * time.Second)
	return nil
}

func resourceVmQemuRead(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	pmParallelBegin(pconf)
	client := pconf.Client
	vmr, err := client.GetVmRefByName(d.Get("name").(string))
	if err != nil {
		return err
	}
	config, err := pxapi.NewConfigQemuFromApi(vmr, client)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}

	d.SetId(resourceId(vmr.Node(), "qemu", vmr.VmId()))
	d.Set("target_node", vmr.Node())
	d.Set("name", config.Name)
	d.Set("desc", config.Description)
	d.Set("onboot", Itob(config.Onboot))
	d.Set("storage", config.Storage)
	d.Set("memory", config.Memory)
	d.Set("cores", config.QemuCores)
	d.Set("sockets", config.QemuSockets)
	d.Set("qemu_os", config.QemuOs)

	// Disks.
	configDisksSet := d.Get("disk").(*schema.Set)
	configDisksMap := devicesSetToMap(configDisksSet)
	activeDisksMap := updateDevicesDefaults(config.QemuDisks, configDisksMap)
	configDisksSetUpdated := updateDevicesSet(configDisksSet, activeDisksMap)
	d.Set("disk", configDisksSetUpdated)

	// Networks.
	configNetworksSet := d.Get("network").(*schema.Set)
	configNetworksMap := devicesSetToMap(configNetworksSet)
	activeNetworksMap := updateDevicesDefaults(config.QemuNetworks, configNetworksMap)
	configNetworksSetUpdated := updateDevicesSet(configNetworksSet, activeNetworksMap)
	d.Set("network", configNetworksSetUpdated)

	// Deprecated.
	d.Set("nic", config.QemuNicModel)
	d.Set("bridge", config.QemuBrige)
	d.Set("vlan", config.QemuVlanTag)
	d.Set("disk_gb", config.DiskSize)

	pmParallelEnd(pconf)
	return nil
}

func resourceVmQemuImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// TODO: research proper import
	err := resourceVmQemuRead(d, meta)
	return []*schema.ResourceData{d}, err
}

func resourceVmQemuDelete(d *schema.ResourceData, meta interface{}) error {
	pconf := meta.(*providerConfiguration)
	pmParallelBegin(pconf)
	client := pconf.Client
	vmId, _ := strconv.Atoi(path.Base(d.Id()))
	vmr := pxapi.NewVmRef(vmId)
	_, err := client.StopVm(vmr)
	if err != nil {
		pmParallelEnd(pconf)
		return err
	}
	// give sometime to proxmox to catchup
	time.Sleep(2 * time.Second)
	_, err = client.DeleteVm(vmr)
	pmParallelEnd(pconf)
	return err
}

func prepareDiskSize(client *pxapi.Client, vmr *pxapi.VmRef, disk_gb float64) error {
	clonedConfig, err := pxapi.NewConfigQemuFromApi(vmr, client)
	if err != nil {
		return err
	}
	if disk_gb > clonedConfig.DiskSize {
		log.Print("[DEBUG] resizing disk")
		_, err = client.ResizeQemuDisk(vmr, "virtio0", int(disk_gb-clonedConfig.DiskSize))
		if err != nil {
			return err
		}
	}
	return nil
}

func devicesSetToMap(devicesSet *schema.Set) pxapi.QemuDevices {

	devicesMap := pxapi.QemuDevices{}

	for _, set := range devicesSet.List() {
		setMap, isMap := set.(pxapi.QemuDevice)
		if isMap {
			setID := setMap["id"].(int)
			devicesMap[setID] = setMap
		}
	}
	return devicesMap
}

func updateDevicesSet(
	devicesSet *schema.Set,
	devicesMap pxapi.QemuDevices,
) *schema.Set {

	for _, setConf := range devicesSet.List() {
		devicesSet.Remove(setConf)
		setConfMap := setConf.(pxapi.QemuDevice)
		deviceID := setConfMap["id"].(int)
		for key, value := range devicesMap[deviceID] {
			// Value type should be one of types allowed by
			// Terraform schema types.
			setConfMap[key] = value
			devicesSet.Add(setConfMap)
		}
	}
	return devicesSet
}

func updateDevicesDefaults(
	activeDevicesMap pxapi.QemuDevices,
	configDevicesMap pxapi.QemuDevices,
) pxapi.QemuDevices {

	for deviceID, deviceConf := range configDevicesMap {
		if _, ok := activeDevicesMap[deviceID]; !ok {
			activeDevicesMap[deviceID] = configDevicesMap[deviceID]
		}
		for key, value := range deviceConf {
			if _, ok := activeDevicesMap[deviceID][key]; !ok {
				activeDevicesMap[deviceID][key] = value
			}
		}
	}
	return activeDevicesMap
}
