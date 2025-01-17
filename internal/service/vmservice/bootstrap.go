/*
Copyright 2023 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vmservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/luthermonson/go-proxmox"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"

	infrav1alpha1 "github.com/ionos-cloud/cluster-api-provider-proxmox/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-proxmox/internal/inject"
	"github.com/ionos-cloud/cluster-api-provider-proxmox/pkg/cloudinit"
	"github.com/ionos-cloud/cluster-api-provider-proxmox/pkg/scope"
)

func reconcileBootstrapData(ctx context.Context, machineScope *scope.MachineScope) (requeue bool, err error) {
	if ptr.Deref(machineScope.ProxmoxMachine.Status.BootstrapDataProvided, false) {
		// skip machine already have the bootstrap data.
		return false, nil
	}

	if !machineHasIPAddress(machineScope.ProxmoxMachine) {
		// skip machine doesn't have an IpAddress yet.
		conditions.MarkFalse(machineScope.ProxmoxMachine, infrav1alpha1.VMProvisionedCondition, infrav1alpha1.WaitingForStaticIPAllocationReason, clusterv1.ConditionSeverityWarning, "no ip address")
		return true, nil
	}

	// make sure MacAddress is set.
	if !vmHasMacAddresses(machineScope) {
		return true, nil
	}

	machineScope.Logger.V(4).Info("reconciling BootstrapData.")

	// Get the bootstrap data.
	bootstrapData, err := getBootstrapData(ctx, machineScope)
	if err != nil {
		conditions.MarkFalse(machineScope.ProxmoxMachine, infrav1alpha1.VMProvisionedCondition, infrav1alpha1.CloningFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return false, err
	}

	biosUUID := extractUUID(machineScope.VirtualMachine.VirtualMachineConfig.SMBios1)

	nicData, err := getNetworkConfigData(ctx, machineScope)
	if err != nil {
		conditions.MarkFalse(machineScope.ProxmoxMachine, infrav1alpha1.VMProvisionedCondition, infrav1alpha1.WaitingForStaticIPAllocationReason, clusterv1.ConditionSeverityWarning, err.Error())
		return false, err
	}

	// create network renderer
	network := cloudinit.NewNetworkConfig(nicData)

	// create metadata renderer
	metadata := cloudinit.NewMetadata(biosUUID, machineScope.Name())

	injector := getISOInjector(machineScope.VirtualMachine, bootstrapData, metadata, network)
	if err = injector.Inject(ctx); err != nil {
		conditions.MarkFalse(machineScope.ProxmoxMachine, infrav1alpha1.VMProvisionedCondition, infrav1alpha1.VMProvisionFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return false, errors.Wrap(err, "cloud-init iso inject failed")
	}

	machineScope.ProxmoxMachine.Status.BootstrapDataProvided = ptr.To(true)

	return false, nil
}

type isoInjector interface {
	Inject(ctx context.Context) error
}

func defaultISOInjector(vm *proxmox.VirtualMachine, bootStrapData []byte, metadata, network cloudinit.Renderer) isoInjector {
	return &inject.ISOInjector{
		VirtualMachine:  vm,
		BootstrapData:   bootStrapData,
		MetaRenderer:    metadata,
		NetworkRenderer: network,
	}
}

var getISOInjector = defaultISOInjector

// getBootstrapData obtains a machine's bootstrap data from the relevant K8s secret and returns the data.
// TODO: Add format return if ignition will be supported.
func getBootstrapData(ctx context.Context, scope *scope.MachineScope) ([]byte, error) {
	if scope.Machine.Spec.Bootstrap.DataSecretName == nil {
		scope.Logger.Info("machine has no bootstrap data.")
		return nil, errors.New("machine has no bootstrap data")
	}

	secret := &corev1.Secret{}
	if err := scope.GetBootstrapSecret(ctx, secret); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve bootstrap data secret")
	}

	value, ok := secret.Data["value"]
	if !ok {
		return nil, errors.New("error retrieving bootstrap data: secret `value` key is missing")
	}

	return value, nil
}

func getNetworkConfigData(ctx context.Context, machineScope *scope.MachineScope) ([]cloudinit.NetworkConfigData, error) {
	// provide a default in case network is not defined
	network := ptr.Deref(machineScope.ProxmoxMachine.Spec.Network, infrav1alpha1.NetworkSpec{})
	networkConfigData := make([]cloudinit.NetworkConfigData, 0, 1+len(network.AdditionalDevices))

	defaultConfig, err := getDefaultNetworkDevice(ctx, machineScope)
	if err != nil {
		return nil, err
	}
	networkConfigData = append(networkConfigData, defaultConfig...)

	additionalConfig, err := getAdditionalNetworkDevices(ctx, machineScope, network)
	if err != nil {
		return nil, err
	}
	networkConfigData = append(networkConfigData, additionalConfig...)

	return networkConfigData, nil
}

func getNetworkConfigDataForDevice(ctx context.Context, machineScope *scope.MachineScope, device string) (*cloudinit.NetworkConfigData, error) {
	nets := machineScope.VirtualMachine.VirtualMachineConfig.MergeNets()
	// For nics supporting multiple IP addresses, we need to cut the '-inet' or '-inet6' part,
	// to retrieve the correct MAC address.
	formattedDevice, _, _ := strings.Cut(device, "-")
	macAddress := extractMACAddress(nets[formattedDevice])
	if len(macAddress) == 0 {
		machineScope.Logger.Error(errors.New("unable to extract mac address"), "device has no mac address", "device", device)
		return nil, errors.New("unable to extract mac address")
	}
	// retrieve IPAddress.
	ipAddr, err := findIPAddress(ctx, machineScope, device)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find IPAddress, device=%s", device)
	}

	dns := machineScope.InfraCluster.ProxmoxCluster.Spec.DNSServers
	ip := IPAddressWithPrefix(ipAddr.Spec.Address, ipAddr.Spec.Prefix)
	gw := ipAddr.Spec.Gateway

	return &cloudinit.NetworkConfigData{
		MacAddress: macAddress,
		IPAddress:  ip,
		Gateway:    gw,
		DNSServers: dns,
	}, nil
}

func getDefaultNetworkDevice(ctx context.Context, machineScope *scope.MachineScope) ([]cloudinit.NetworkConfigData, error) {
	var config cloudinit.NetworkConfigData

	// default network device ipv4.
	if machineScope.InfraCluster.ProxmoxCluster.Spec.IPv4Config != nil {
		conf, err := getNetworkConfigDataForDevice(ctx, machineScope, DefaultNetworkDeviceIPV4)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get network config data for device=%s", DefaultNetworkDeviceIPV4)
		}
		config = *conf
	}

	// default network device ipv6.
	if machineScope.InfraCluster.ProxmoxCluster.Spec.IPv6Config != nil {
		conf, err := getNetworkConfigDataForDevice(ctx, machineScope, DefaultNetworkDeviceIPV6)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get network config data for device=%s", DefaultNetworkDeviceIPV6)
		}

		switch {
		case len(config.MacAddress) == 0:
			config = *conf
		case config.MacAddress != conf.MacAddress:
			return nil, errors.New("default network device ipv4 and ipv6 have different mac addresses")
		default:
			config.IPV6Address = conf.IPAddress
			config.Gateway6 = conf.Gateway
		}
	}

	return []cloudinit.NetworkConfigData{config}, nil
}

func getAdditionalNetworkDevices(ctx context.Context, machineScope *scope.MachineScope, network infrav1alpha1.NetworkSpec) ([]cloudinit.NetworkConfigData, error) {
	networkConfigData := make([]cloudinit.NetworkConfigData, 0, len(network.AdditionalDevices))

	// additional network devices.
	for _, nic := range network.AdditionalDevices {
		var config = ptr.To(cloudinit.NetworkConfigData{})

		if nic.IPv4PoolRef != nil {
			device := fmt.Sprintf("%s-%s", nic.Name, infrav1alpha1.DefaultSuffix)
			conf, err := getNetworkConfigDataForDevice(ctx, machineScope, device)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to get network config data for device=%s", device)
			}
			if len(nic.DNSServers) != 0 {
				config.DNSServers = nic.DNSServers
			}
			config = conf
		}

		if nic.IPv6PoolRef != nil {
			suffix := infrav1alpha1.DefaultSuffix + "6"
			device := fmt.Sprintf("%s-%s", nic.Name, suffix)
			conf, err := getNetworkConfigDataForDevice(ctx, machineScope, device)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to get network config data for device=%s", device)
			}
			if len(nic.DNSServers) != 0 {
				config.DNSServers = nic.DNSServers
			}

			switch {
			case len(config.MacAddress) == 0:
				config = conf
			case config.MacAddress != conf.MacAddress:
				return nil, errors.New("additional network device ipv4 and ipv6 have different mac addresses")
			default:
				config.IPV6Address = conf.IPAddress
				config.Gateway6 = conf.Gateway
			}
		}

		if len(config.MacAddress) > 0 {
			networkConfigData = append(networkConfigData, *config)
		}
	}
	return networkConfigData, nil
}

func vmHasMacAddresses(machineScope *scope.MachineScope) bool {
	nets := machineScope.VirtualMachine.VirtualMachineConfig.MergeNets()
	if len(nets) == 0 {
		return false
	}
	for d := range nets {
		if macAddress := extractMACAddress(nets[d]); macAddress == "" {
			return false
		}
	}
	return true
}
