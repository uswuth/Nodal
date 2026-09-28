package dns

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"nodal/src/config"
	"nodal/src/winapi"

	"golang.org/x/sys/windows"
)

var (
	ErrNoActiveAdapter = errors.New("no active network adapter with default gateway found")
)

const (
	NetTypeEthernet uint32 = 6
	NetTypeWiFi     uint32 = 71
	NetTypeVPN      uint32 = 131
)

type AdapterInfo struct {
	GUID         windows.GUID
	GUIDString   string
	FriendlyName string
	Description  string
	SSID         string
	IfType       uint32
	Metric       uint32
	DNSServers   []string
}

func (a *AdapterInfo) IsVPN() bool {
	lowerDesc := strings.ToLower(a.Description)
	lowerName := strings.ToLower(a.FriendlyName)
	return a.IfType == 23 || a.IfType == 131 ||
		strings.Contains(lowerDesc, "vpn") || strings.Contains(lowerName, "vpn") ||
		strings.Contains(lowerDesc, "wireguard") || strings.Contains(lowerName, "wireguard") ||
		strings.Contains(lowerDesc, "tap-") || strings.Contains(lowerDesc, "wintun") ||
		strings.Contains(lowerDesc, "tailscale") || strings.Contains(lowerName, "tailscale")
}

func (a *AdapterInfo) DisplayName() string {
	// 1. Wi-Fi: show connected network SSID
	if a.SSID != "" {
		return a.SSID
	}
	// 2. VPN: show connection name
	if a.IsVPN() {
		return a.FriendlyName
	}
	// 3. Ethernet: show connection friendly name (e.g. "Ethernet")
	if a.FriendlyName != "" {
		return a.FriendlyName
	}
	return "Connected"
}

var (
	wlanapi                = windows.NewLazyDLL("wlanapi.dll")
	procWlanOpenHandle     = wlanapi.NewProc("WlanOpenHandle")
	procWlanCloseHandle    = wlanapi.NewProc("WlanCloseHandle")
	procWlanEnumInterfaces = wlanapi.NewProc("WlanEnumInterfaces")
	procWlanQueryInterface = wlanapi.NewProc("WlanQueryInterface")
	procWlanFreeMemory     = wlanapi.NewProc("WlanFreeMemory")
)

type wlanInterfaceInfo struct {
	InterfaceGuid            windows.GUID
	strInterfaceDescription [256]uint16
	isState                  uint32
}

type wlanInterfaceInfoList struct {
	dwNumberOfItems uint32
	dwIndex         uint32
	InterfaceInfo   [1]wlanInterfaceInfo
}

type dot11SSID struct {
	uSSIDLength uint32
	ucSSID      [32]byte
}

type wlanAssociationAttributes struct {
	Dot11Ssid         dot11SSID
	Dot11BssType      uint32
	Dot11Bssid        [6]byte
	Dot11PhyType      uint32
	UDot11PhyIndex    uint32
	WlanSignalQuality uint32
	UlRxRate          uint32
	UlTxRate          uint32
}

type wlanConnectionAttributes struct {
	IsState                   uint32
	WlanConnectionMode        uint32
	StrProfileName            [256]uint16
	WlanAssociationAttributes wlanAssociationAttributes
}

// GetNativeWifiSSID queries wlanapi.dll directly for the connected Wi-Fi SSID
func GetNativeWifiSSID() string {
	if wlanapi.Load() != nil {
		return ""
	}

	var negotiatedVersion uint32
	var clientHandle windows.Handle
	r1, _, _ := procWlanOpenHandle.Call(2, 0, uintptr(unsafe.Pointer(&negotiatedVersion)), uintptr(unsafe.Pointer(&clientHandle)))
	if r1 != 0 {
		return ""
	}
	defer procWlanCloseHandle.Call(uintptr(clientHandle), 0)

	var pIfList *wlanInterfaceInfoList
	r1, _, _ = procWlanEnumInterfaces.Call(uintptr(clientHandle), 0, uintptr(unsafe.Pointer(&pIfList)))
	if r1 != 0 || pIfList == nil {
		return ""
	}
	defer procWlanFreeMemory.Call(uintptr(unsafe.Pointer(pIfList)))

	numIfs := pIfList.dwNumberOfItems
	if numIfs == 0 {
		return ""
	}

	pInfoSlice := (*[1 << 16]wlanInterfaceInfo)(unsafe.Pointer(&pIfList.InterfaceInfo[0]))[:numIfs:numIfs]
	for i := uint32(0); i < numIfs; i++ {
		info := pInfoSlice[i]
		var dataSize uint32
		var pConnAttr *wlanConnectionAttributes
		var opcodeType uint32
		r1, _, _ = procWlanQueryInterface.Call(
			uintptr(clientHandle),
			uintptr(unsafe.Pointer(&info.InterfaceGuid)),
			7, // wlan_intf_opcode_current_connection
			0,
			uintptr(unsafe.Pointer(&dataSize)),
			uintptr(unsafe.Pointer(&pConnAttr)),
			uintptr(unsafe.Pointer(&opcodeType)),
		)
		if r1 == 0 && pConnAttr != nil {
			defer procWlanFreeMemory.Call(uintptr(unsafe.Pointer(pConnAttr)))

			// 1. Check dot11Ssid
			ssidLen := pConnAttr.WlanAssociationAttributes.Dot11Ssid.uSSIDLength
			if ssidLen > 0 && ssidLen <= 32 {
				ssid := string(pConnAttr.WlanAssociationAttributes.Dot11Ssid.ucSSID[:ssidLen])
				if strings.TrimSpace(ssid) != "" {
					return strings.TrimSpace(ssid)
				}
			}

			// 2. Fallback to profile name
			profileName := windows.UTF16ToString(pConnAttr.StrProfileName[:])
			if strings.TrimSpace(profileName) != "" {
				return strings.TrimSpace(profileName)
			}
		}
	}
	return ""
}

// GetWifiSSID retrieves the connected Wi-Fi network SSID
func GetWifiSSID() string {
	if ssid := GetNativeWifiSSID(); ssid != "" {
		return ssid
	}
	// Fallback to netsh
	cmd := exec.Command("netsh.exe", "wlan", "show", "interfaces")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if (strings.HasPrefix(trimmed, "SSID") && !strings.HasPrefix(trimmed, "BSSID")) || strings.HasPrefix(trimmed, "Profile") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" {
					return val
				}
			}
		}
	}
	return ""
}


// GetActiveAdapter detects the primary connected network adapter (lowest metric with active gateway).
// Fulfills §9 edge cases: distinguishes multiple default routes and handles disconnected state.
func GetActiveAdapter() (*AdapterInfo, error) {
	const (
		GAA_FLAG_INCLUDE_GATEWAYS = 0x0080
		IF_TYPE_SOFTWARE_LOOPBACK = 24
		IfOperStatusUp            = 1
	)

	var size uint32 = 16384
	for {
		buf := make([]byte, size)
		pAddresses := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err := windows.GetAdaptersAddresses(
			windows.AF_INET,
			GAA_FLAG_INCLUDE_GATEWAYS,
			0,
			pAddresses,
			&size,
		)
		if err == nil {
			var bestAdapter *AdapterInfo
			var lowestMetric uint32 = ^uint32(0)

			for curr := pAddresses; curr != nil; curr = curr.Next {
				// Must be operational and not loopback
				if curr.OperStatus != IfOperStatusUp || curr.IfType == IF_TYPE_SOFTWARE_LOOPBACK {
					continue
				}

				// Must have a gateway to be the active route carrying traffic
				if curr.FirstGatewayAddress == nil {
					continue
				}

				metric := curr.Ipv4Metric
				if bestAdapter == nil || metric < lowestMetric {
					guidStr := windows.BytePtrToString(curr.AdapterName)
					guid, err := windows.GUIDFromString(guidStr)
					if err != nil {
						continue
					}

					// Collect configured DNS servers
					var dnsList []string
					for dnsNode := curr.FirstDnsServerAddress; dnsNode != nil; dnsNode = dnsNode.Next {
						if dnsNode.Address.Sockaddr != nil {
							sa := (*windows.RawSockaddr)(unsafe.Pointer(dnsNode.Address.Sockaddr))
							if sa.Family == windows.AF_INET {
								sa4 := (*windows.RawSockaddrInet4)(unsafe.Pointer(dnsNode.Address.Sockaddr))
								ip := net.IPv4(sa4.Addr[0], sa4.Addr[1], sa4.Addr[2], sa4.Addr[3])
								dnsList = append(dnsList, ip.String())
							}
						}
					}

					lowestMetric = metric
					bestAdapter = &AdapterInfo{
						GUID:         guid,
						GUIDString:   guidStr,
						FriendlyName: windows.UTF16PtrToString(curr.FriendlyName),
						Description:  windows.UTF16PtrToString(curr.Description),
						IfType:       curr.IfType,
						Metric:       metric,
						DNSServers:   dnsList,
					}
				}
			}

			if bestAdapter != nil {
				// Check for active Wi-Fi SSID
				if ssid := GetWifiSSID(); ssid != "" {
					bestAdapter.SSID = ssid
					bestAdapter.IfType = NetTypeWiFi
				} else if bestAdapter.IsVPN() {
					bestAdapter.IfType = NetTypeVPN
				}
			}



			if bestAdapter == nil {
				return nil, ErrNoActiveAdapter
			}
			return bestAdapter, nil
		}

		if errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			size *= 2
			continue
		}
		return nil, fmt.Errorf("failed to query network adapters: %w", err)
	}
}

// ApplyProfile sets the DNS configuration on the active adapter using SetInterfaceDnsSettings.
// If profile is DHCP (empty servers), it clears static DNS.
func ApplyProfile(profile config.DNSProfile) error {
	adapter, err := GetActiveAdapter()
	if err != nil {
		return err
	}

	var settings winapi.DNS_INTERFACE_SETTINGS
	settings.Version = winapi.DNS_INTERFACE_SETTINGS_VERSION1
	settings.Flags = winapi.DNS_SETTING_NAMESERVER

	var servers []string
	if strings.TrimSpace(profile.Primary) != "" {
		servers = append(servers, strings.TrimSpace(profile.Primary))
	}
	if strings.TrimSpace(profile.Secondary) != "" {
		servers = append(servers, strings.TrimSpace(profile.Secondary))
	}

	if len(servers) > 0 {
		nsStr := strings.Join(servers, ",")
		pNs, err := windows.UTF16PtrFromString(nsStr)
		if err != nil {
			return fmt.Errorf("invalid DNS servers string: %w", err)
		}
		settings.NameServer = pNs
	} else {
		// Empty NameServer clears static override, reverting interface to DHCP
		settings.NameServer = nil
	}

	res := winapi.SetInterfaceDnsSettings(&adapter.GUID, &settings)
	if res != 0 {
		return fmt.Errorf("SetInterfaceDnsSettings failed with code 0x%X (error %d)", res, res)
	}

	// Also invoke netsh to ensure Windows Settings GUI and NLA reflect the change immediately
	if len(servers) > 0 {
		cmd1 := exec.Command("netsh.exe", "interface", "ipv4", "set", "dnsservers", fmt.Sprintf("name=%s", adapter.FriendlyName), "source=static", fmt.Sprintf("address=%s", servers[0]), "validate=no")
		cmd1.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd1.Run()

		if len(servers) > 1 {
			cmd2 := exec.Command("netsh.exe", "interface", "ipv4", "add", "dnsservers", fmt.Sprintf("name=%s", adapter.FriendlyName), fmt.Sprintf("address=%s", servers[1]), "index=2", "validate=no")
			cmd2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = cmd2.Run()
		}
	} else {
		cmd := exec.Command("netsh.exe", "interface", "ipv4", "set", "dnsservers", fmt.Sprintf("name=%s", adapter.FriendlyName), "source=dhcp")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run()
	}

	// Apply IPv6 DNS if configured
	if strings.TrimSpace(profile.PrimaryV6) != "" {
		cmdV6_1 := exec.Command("netsh.exe", "interface", "ipv6", "set", "dnsservers", fmt.Sprintf("name=%s", adapter.FriendlyName), "source=static", fmt.Sprintf("address=%s", strings.TrimSpace(profile.PrimaryV6)), "validate=no")
		cmdV6_1.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmdV6_1.Run()

		if strings.TrimSpace(profile.SecondaryV6) != "" {
			cmdV6_2 := exec.Command("netsh.exe", "interface", "ipv6", "add", "dnsservers", fmt.Sprintf("name=%s", adapter.FriendlyName), fmt.Sprintf("address=%s", strings.TrimSpace(profile.SecondaryV6)), "index=2", "validate=no")
			cmdV6_2.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = cmdV6_2.Run()
		}
	} else if len(servers) == 0 {
		// If switching to DHCP, also clear IPv6 overrides
		cmdV6 := exec.Command("netsh.exe", "interface", "ipv6", "set", "dnsservers", fmt.Sprintf("name=%s", adapter.FriendlyName), "source=dhcp")
		cmdV6.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmdV6.Run()
	}

	// Flush DNS cache
	winapi.DnsFlushResolverCache()
	return nil
}

// GetAllAdapters lists every IPv4 interface except loopback, together with the DNS servers
// currently assigned to it. Unlike GetActiveAdapter it does not pick a single winner, so it also
// covers adapters that are currently disconnected but still carry a static override.
func GetAllAdapters() ([]*AdapterInfo, error) {
	const (
		GAA_FLAG_INCLUDE_GATEWAYS = 0x0080
		IF_TYPE_SOFTWARE_LOOPBACK = 24
	)

	var size uint32 = 16384
	for {
		buf := make([]byte, size)
		pAddresses := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
		err := windows.GetAdaptersAddresses(
			windows.AF_INET,
			GAA_FLAG_INCLUDE_GATEWAYS,
			0,
			pAddresses,
			&size,
		)
		if err == nil {
			var adapters []*AdapterInfo
			for curr := pAddresses; curr != nil; curr = curr.Next {
				if curr.IfType == IF_TYPE_SOFTWARE_LOOPBACK {
					continue
				}

				guidStr := windows.BytePtrToString(curr.AdapterName)
				guid, err := windows.GUIDFromString(guidStr)
				if err != nil {
					continue
				}

				var dnsList []string
				for dnsNode := curr.FirstDnsServerAddress; dnsNode != nil; dnsNode = dnsNode.Next {
					if dnsNode.Address.Sockaddr != nil {
						sa := (*windows.RawSockaddr)(unsafe.Pointer(dnsNode.Address.Sockaddr))
						if sa.Family == windows.AF_INET {
							sa4 := (*windows.RawSockaddrInet4)(unsafe.Pointer(dnsNode.Address.Sockaddr))
							ip := net.IPv4(sa4.Addr[0], sa4.Addr[1], sa4.Addr[2], sa4.Addr[3])
							dnsList = append(dnsList, ip.String())
						}
					}
				}

				adapters = append(adapters, &AdapterInfo{
					GUID:         guid,
					GUIDString:   guidStr,
					FriendlyName: windows.UTF16PtrToString(curr.FriendlyName),
					Description:  windows.UTF16PtrToString(curr.Description),
					IfType:       curr.IfType,
					Metric:       curr.Ipv4Metric,
					DNSServers:   dnsList,
				})
			}
			return adapters, nil
		}

		if errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			size *= 2
			continue
		}
		return nil, fmt.Errorf("failed to query network adapters: %w", err)
	}
}

// ResetAllToDHCP clears the static DNS override on every adapter so Windows hands the resolvers
// back to DHCP. Used by the uninstaller to leave the machine as it was before Nodal was installed.
func ResetAllToDHCP() error {
	adapters, err := GetAllAdapters()
	if err != nil {
		return err
	}

	var failures []string
	for _, adapter := range adapters {
		if err := resetAdapterToDHCP(adapter); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", adapter.DisplayName(), err))
		}
	}

	winapi.DnsFlushResolverCache()

	if len(failures) > 0 {
		return fmt.Errorf("failed to restore automatic DNS on: %s", strings.Join(failures, "; "))
	}
	return nil
}

func resetAdapterToDHCP(adapter *AdapterInfo) error {
	var settings winapi.DNS_INTERFACE_SETTINGS
	settings.Version = winapi.DNS_INTERFACE_SETTINGS_VERSION1
	settings.Flags = winapi.DNS_SETTING_NAMESERVER
	// A nil NameServer clears the static override, reverting the interface to DHCP.
	settings.NameServer = nil

	if res := winapi.SetInterfaceDnsSettings(&adapter.GUID, &settings); res != 0 {
		return fmt.Errorf("SetInterfaceDnsSettings failed with code 0x%X (error %d)", res, res)
	}

	// Mirror the change in the Windows Settings GUI and drop any IPv6 override as well.
	for _, family := range []string{"ipv4", "ipv6"} {
		cmd := exec.Command("netsh.exe", "interface", family, "set", "dnsservers", fmt.Sprintf("name=%s", adapter.FriendlyName), "source=dhcp")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run()
	}
	return nil
}

// DetectCurrentProfile matches current active adapter DNS servers with defined profiles
func DetectCurrentProfile(profiles []config.DNSProfile) string {
	adapter, err := GetActiveAdapter()
	if err != nil || len(adapter.DNSServers) == 0 {
		return "DHCP"
	}

	for _, p := range profiles {
		if strings.TrimSpace(p.Primary) == "" {
			continue
		}
		// Match primary IP
		for _, srv := range adapter.DNSServers {
			if srv == strings.TrimSpace(p.Primary) {
				return p.Name
			}
		}
	}

	return "DHCP"
}
