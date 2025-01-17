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

package goproxmox

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/jarcoal/httpmock"
	"github.com/luthermonson/go-proxmox"
	"github.com/stretchr/testify/require"
)

const testBaseURL = "http://pve.local.test/" // regression test against trailing /

func newTestClient(t *testing.T) *APIClient {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	httpmock.RegisterResponder(http.MethodGet, testBaseURL+"api2/json/version",
		newJSONResponder(200, proxmox.Version{Release: "test"}))

	client, err := NewAPIClient(context.Background(), logr.Discard(), testBaseURL)
	require.NoError(t, err)

	return client
}

func newJSONResponder(status int, data any) httpmock.Responder {
	return httpmock.NewJsonResponderOrPanic(status, map[string]any{"data": data}).Once()
}

func TestProxmoxAPIClient_GetReservableMemoryBytes(t *testing.T) {
	tests := []struct {
		name   string
		maxMem uint64
		expect uint64
	}{
		{name: "under zero", maxMem: 29, expect: 1},
		{name: "exact zero", maxMem: 30, expect: 0},
		{name: "over zero", maxMem: 31, expect: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(t)
			httpmock.RegisterResponder(http.MethodGet, `=~/nodes/test/status`,
				newJSONResponder(200, proxmox.Node{Memory: proxmox.Memory{Total: 30}}))

			httpmock.RegisterResponder(http.MethodGet, `=~/nodes/test/qemu`,
				newJSONResponder(200, proxmox.VirtualMachines{{MaxMem: test.maxMem}}))

			reservable, err := client.GetReservableMemoryBytes(context.Background(), "test")
			require.NoError(t, err)
			require.Equal(t, test.expect, reservable)
		})
	}
}
