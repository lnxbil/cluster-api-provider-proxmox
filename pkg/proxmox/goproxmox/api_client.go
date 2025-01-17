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

// Package goproxmox implements a client for Proxmox resource lifecycle management.
package goproxmox

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/luthermonson/go-proxmox"

	capmox "github.com/ionos-cloud/cluster-api-provider-proxmox/pkg/proxmox"
)

var _ capmox.Client = &APIClient{}

// APIClient Proxmox API client object.
type APIClient struct {
	*proxmox.Client
	logger logr.Logger
}

// NewAPIClient initializes a Proxmox API client. If the client is misconfigured, an error is returned.
func NewAPIClient(ctx context.Context, logger logr.Logger, baseURL string, options ...proxmox.Option) (*APIClient, error) {
	proxmoxAPIURL, err := url.JoinPath(baseURL, "api2", "json")
	if err != nil {
		return nil, fmt.Errorf("invalid proxmox base URL %q: %w", baseURL, err)
	}

	options = append(options, proxmox.WithLogger(capmox.Logger{}))
	upstreamClient := proxmox.NewClient(proxmoxAPIURL, options...)
	version, err := upstreamClient.Version(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize proxmox api client: %w", err)
	}
	logger.Info("Proxmox client initialized")
	logger.Info("Proxmox server", "version", version.Release)

	return &APIClient{
		Client: upstreamClient,
		logger: logger,
	}, nil
}

// CloneVM clones a VM based on templateID and VMCloneRequest.
func (c *APIClient) CloneVM(ctx context.Context, templateID int, clone capmox.VMCloneRequest) (capmox.VMCloneResponse, error) {
	// get the node
	node, err := c.Node(ctx, clone.Node)
	if err != nil {
		return capmox.VMCloneResponse{}, fmt.Errorf("cannot find node with name %s: %w", clone.Node, err)
	}

	// get the vm template
	vmTemplate, err := node.VirtualMachine(ctx, templateID)
	if err != nil {
		return capmox.VMCloneResponse{}, fmt.Errorf("unable to find vm template: %w", err)
	}

	vmOptions := proxmox.VirtualMachineCloneOptions{
		NewID:       clone.NewID,
		Description: clone.Description,
		Format:      clone.Format,
		Full:        clone.Full,
		Name:        clone.Name,
		Pool:        clone.Pool,
		SnapName:    clone.SnapName,
		Storage:     clone.Storage,
		Target:      clone.Target,
	}
	newID, task, err := vmTemplate.Clone(ctx, &vmOptions)
	if err != nil {
		return capmox.VMCloneResponse{}, fmt.Errorf("unable to create new vm: %w", err)
	}

	return capmox.VMCloneResponse{NewID: int64(newID), Task: task}, nil
}

// ConfigureVM updates a VMs settings.
func (c *APIClient) ConfigureVM(ctx context.Context, vm *proxmox.VirtualMachine, options ...capmox.VirtualMachineOption) (*proxmox.Task, error) {
	task, err := vm.Config(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("unable to configure vm: %w", err)
	}
	return task, nil
}

// GetVM returns a VM based on nodeName and vmID.
func (c *APIClient) GetVM(ctx context.Context, nodeName string, vmID int64) (*proxmox.VirtualMachine, error) {
	node, err := c.Node(ctx, nodeName)
	if err != nil {
		return nil, fmt.Errorf("cannot find node with name %s: %w", nodeName, err)
	}

	vm, err := node.VirtualMachine(ctx, int(vmID))
	if err != nil {
		return nil, fmt.Errorf("cannot find vm with id %d: %w", vmID, err)
	}

	return vm, nil
}

// FindVMResource tries to find a VM by its ID on the whole cluster.
func (c *APIClient) FindVMResource(ctx context.Context, vmID uint64) (*proxmox.ClusterResource, error) {
	cluster, err := c.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get cluster status: %w", err)
	}

	vmResources, err := cluster.Resources(ctx, "vm")
	if err != nil {
		return nil, fmt.Errorf("could not list vm resources: %w", err)
	}

	for _, vm := range vmResources {
		if vm.VMID == vmID {
			return vm, nil
		}
	}

	return nil, fmt.Errorf("unable to find VM with ID %d on any of the nodes", vmID)
}

// DeleteVM deletes a VM based on the nodeName and vmID.
func (c *APIClient) DeleteVM(ctx context.Context, nodeName string, vmID int64) (*proxmox.Task, error) {
	node, err := c.Node(ctx, nodeName)
	if err != nil {
		return nil, fmt.Errorf("cannot find node with name %s: %w", nodeName, err)
	}

	vm, err := node.VirtualMachine(ctx, int(vmID))
	if err != nil {
		return nil, fmt.Errorf("cannot find vm with id %d: %w", vmID, err)
	}

	if vm.IsRunning() {
		if _, err = vm.Stop(ctx); err != nil {
			return nil, fmt.Errorf("cannot stop vm id %d: %w", vmID, err)
		}
	}

	task, err := vm.Delete(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot delete vm with id %d: %w", vmID, err)
	}

	return task, nil
}

// GetTask returns a task associated with upID.
func (c *APIClient) GetTask(ctx context.Context, upID string) (*proxmox.Task, error) {
	task := proxmox.NewTask(proxmox.UPID(upID), c.Client)

	err := task.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get task with UPID %s: %w", upID, err)
	}

	return task, nil
}

// GetReservableMemoryBytes returns the memory that can be reserved by a new VM, in bytes.
func (c *APIClient) GetReservableMemoryBytes(ctx context.Context, nodeName string) (uint64, error) {
	node, err := c.Client.Node(ctx, nodeName)
	if err != nil {
		return 0, fmt.Errorf("cannot find node with name %s: %w", nodeName, err)
	}

	reservableMemory := node.Memory.Total

	vms, err := node.VirtualMachines(ctx)
	if err != nil {
		return 0, fmt.Errorf("cannot list vms for node %s: %w", nodeName, err)
	}

	for _, vm := range vms {
		if reservableMemory < vm.MaxMem {
			reservableMemory = 0
		} else {
			reservableMemory -= vm.MaxMem
		}
	}

	return reservableMemory, nil
}

// ResizeDisk resizes a VM disk to the specified size.
func (c *APIClient) ResizeDisk(ctx context.Context, vm *proxmox.VirtualMachine, disk, size string) error {
	return vm.ResizeDisk(ctx, disk, size)
}

// ResumeVM resumes the VM.
func (c *APIClient) ResumeVM(ctx context.Context, vm *proxmox.VirtualMachine) (*proxmox.Task, error) {
	return vm.Resume(ctx)
}

// StartVM starts the VM.
func (c *APIClient) StartVM(ctx context.Context, vm *proxmox.VirtualMachine) (*proxmox.Task, error) {
	return vm.Start(ctx)
}

// TagVM tags the VM.
func (c *APIClient) TagVM(ctx context.Context, vm *proxmox.VirtualMachine, tag string) (*proxmox.Task, error) {
	return vm.AddTag(ctx, tag)
}
