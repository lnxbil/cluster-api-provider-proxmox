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

// NOSONAR
package cloudinit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	expectedValidNetworkConfig = `network:
  version: 2
  renderer: networkd
  ethernets:
    eth0:
      match:
        macaddress: 92:60:a0:5b:22:c2
      dhcp4: 'no'
      addresses:
        - 10.10.10.12/24
      routes:
        - to: default
          via: 10.10.10.1
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4`

	expectedValidNetworkConfigWithoutDNS = `network:
  version: 2
  renderer: networkd
  ethernets:
    eth0:
      match:
        macaddress: 92:60:a0:5b:22:c2
      dhcp4: 'no'
      addresses:
        - 10.10.10.12/24
      routes:
        - to: default
          via: 10.10.10.1`

	expectedValidNetworkConfigMultipleNics = `network:
  version: 2
  renderer: networkd
  ethernets:
    eth0:
      match:
        macaddress: 92:60:a0:5b:22:c2
      dhcp4: 'no'
      addresses:
        - 10.10.10.12/24
      routes:
        - to: default
          via: 10.10.10.1
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4
    eth1:
      match:
        macaddress: b4:87:18:bf:a3:60
      dhcp4: 'no'
      addresses:
        - 196.168.100.124/24
      routes:
        - to: default
          via: 196.168.100.254
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4`

	expectedValidNetworkConfigDualStack = `network:
  version: 2
  renderer: networkd
  ethernets:
    eth0:
      match:
        macaddress: 92:60:a0:5b:22:c2
      dhcp4: 'no'
      addresses:
        - 10.10.10.12/24
        - 2001:db8::1/64
      routes:
        - to: default
          via: 10.10.10.1
        - to: default
          via: 2001:db8::1
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4`

	expectedValidNetworkConfigIPV6 = `network:
  version: 2
  renderer: networkd
  ethernets:
    eth0:
      match:
        macaddress: 92:60:a0:5b:22:c2
      dhcp4: 'no'
      addresses:
        - 2001:db8::1/64
      routes:
        - to: default
          via: 2001:db8::1
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4`
)

func TestNetworkConfig_Render(t *testing.T) {
	type args struct {
		nics []NetworkConfigData
	}

	type want struct {
		network string
		err     error
	}

	cases := map[string]struct {
		reason string
		args   args
		want   want
	}{
		"ValidNetworkConfig": {
			reason: "render valid network-config",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress: "92:60:a0:5b:22:c2",
						IPAddress:  "10.10.10.12/24",
						Gateway:    "10.10.10.1",
						DNSServers: []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: expectedValidNetworkConfig,
				err:     nil,
			},
		},
		"InvalidNetworkConfigIp": {
			reason: "ip address is not set",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress: "92:60:a0:5b:22:c2",
						Gateway:    "10.10.10.1",
						DNSServers: []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: "",
				err:     ErrMissingIPAddress,
			},
		},
		"InvalidNetworkConfigMalformedIp": {
			reason: "malformed ip address",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress: "92:60:a0:5b:22:c2",
						IPAddress:  "10.10.10.12",
						Gateway:    "10.10.10.1",
						DNSServers: []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: "",
				err:     ErrMalformedIPAddress,
			},
		},
		"InvalidNetworkConfigMalformedIP": {
			reason: "ip address malformed",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress: "92:60:a0:5b:22:c2",
						IPAddress:  "10.10.10.115",
						Gateway:    "10.10.10.1",
						DNSServers: []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: "",
				err:     ErrMalformedIPAddress,
			},
		},
		"InvalidNetworkConfigGW": {
			reason: "gw is not set",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress: "92:60:a0:5b:22:c2",
						IPAddress:  "10.10.10.12/24",
						DNSServers: []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: "",
				err:     ErrMissingGateway,
			},
		},
		"InvalidNetworkConfigMacAddress": {
			reason: "macaddress is not set",
			args: args{
				nics: []NetworkConfigData{
					{
						IPAddress:  "10.10.10.11/24",
						Gateway:    "10.10.10.1",
						DNSServers: []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: "",
				err:     ErrMissingMacAddress,
			},
		},
		"ValidNetworkConfigWithoutDNS": {
			reason: "valid config without dns",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress: "92:60:a0:5b:22:c2",
						IPAddress:  "10.10.10.12/24",
						Gateway:    "10.10.10.1",
					},
				},
			},
			want: want{
				network: expectedValidNetworkConfigWithoutDNS,
				err:     nil,
			},
		},
		"ValidNetworkConfigMultipleNics": {
			reason: "valid config multiple nics",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress: "92:60:a0:5b:22:c2",
						IPAddress:  "10.10.10.12/24",
						Gateway:    "10.10.10.1",
						DNSServers: []string{"8.8.8.8", "8.8.4.4"},
					},
					{
						MacAddress: "b4:87:18:bf:a3:60",
						IPAddress:  "196.168.100.124/24",
						Gateway:    "196.168.100.254",
						DNSServers: []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: expectedValidNetworkConfigMultipleNics,
				err:     nil,
			},
		},
		"InvalidNetworkConfigData": {
			reason: "invalid config missing network config data",
			args: args{
				nics: []NetworkConfigData{},
			},
			want: want{
				network: "",
				err:     ErrMissingNetworkConfigData,
			},
		},
		"ValidNetworkConfigDualStack": {
			reason: "render valid network-config",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress:  "92:60:a0:5b:22:c2",
						IPAddress:   "10.10.10.12/24",
						IPV6Address: "2001:db8::1/64",
						Gateway6:    "2001:db8::1",
						Gateway:     "10.10.10.1",
						DNSServers:  []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: expectedValidNetworkConfigDualStack,
				err:     nil,
			},
		},
		"ValidNetworkConfigIPV6": {
			reason: "render valid ipv6 network-config",
			args: args{
				nics: []NetworkConfigData{
					{
						MacAddress:  "92:60:a0:5b:22:c2",
						IPV6Address: "2001:db8::1/64",
						Gateway6:    "2001:db8::1",
						DNSServers:  []string{"8.8.8.8", "8.8.4.4"},
					},
				},
			},
			want: want{
				network: expectedValidNetworkConfigIPV6,
				err:     nil,
			},
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			nc := NewNetworkConfig(tc.args.nics)
			network, err := nc.Render()
			require.ErrorIs(t, err, tc.want.err)
			require.Equal(t, tc.want.network, string(network))
		})
	}
}
