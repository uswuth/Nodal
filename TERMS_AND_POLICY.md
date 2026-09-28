# Terms of Service & End User License Agreement (EULA)

**Last Updated:** September 2026

| Product identity | Value |
| --- | --- |
| Application name | Nodal |
| Product description | Native Windows DNS Switcher (system tray utility) |
| Publisher / Vendor | Nodal Open Source Project |
| Publisher website / source repository | https://github.com/nodal/dns-switcher |
| Support & bug reports | https://github.com/nodal/dns-switcher/issues |
| Product version | 1.0.0 |
| Price | Free of charge — no cost, no subscription, no trial, no license key |
| License | MIT License (Free and Open Source Software) |
| Supported platform | Windows 10 / Windows 11, 64-bit (x64 or Arm64 with x64 emulation) |

---

### 1. Acceptance of Terms
By downloading, installing, copying, or using Nodal ("Software"), you agree to be bound by the terms and conditions set forth in this Agreement. If you do not agree to these terms, do not download, install, or use the Software.

### 2. Free and Open Source Software (FOSS)
Nodal is provided 100% free of charge. You may inspect, modify, compile, and distribute the source code in accordance with the MIT License. No fees, subscriptions, or payments are required to use any feature of this Software.

### 3. Application Scope and System Rights Under Law
To operate correctly and fulfill its sole function, Nodal requests and uses the following system capabilities:
- **Network Adapter Settings Modification (`SetInterfaceDnsSettings` / `netsh`):** Used strictly to update the primary/secondary IPv4 and IPv6 DNS resolver addresses on your active network adapter when you select a DNS profile.
- **Task Scheduler (`schtasks.exe`):** Creates an on-demand elevated worker task (`Nodal`) so you can switch DNS servers securely without repeated Windows User Account Control (UAC) prompts.
- **Startup Entry (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`):** An optional registry entry used exclusively if you enable the "autostart" feature in your configuration.
- **Local Configuration Storage (`%USERPROFILE%\.config\nodal\`):** Stores your personal DNS profile presets, UI display mode, and settings locally in plain text (`config.toml`).

Uninstalling reverses every one of these changes: the static DNS resolvers are replaced by the automatically assigned (DHCP) resolvers on every adapter, the privileged worker task and the optional startup entry are deleted, the settings folders above are removed, and then the application itself is uninstalled. Only objects created by Nodal are touched.

Nodal does **not**:
- Act as a VPN, proxy, or tunnel.
- Route, inspect, intercept, or modify your network payload or web traffic.
- Inject third-party drivers, background services, or network filter extensions (WFP).
- Run background analytics, telemetries, or remote network listeners.

### 4. Consumer Data Protection, Safety & Privacy
- **Zero Data Collection:** Nodal does not collect, record, log, transmit, sell, or share any personal identifiable information (PII), browsing history, DNS queries, or system diagnostics.
- **Local Operation:** All DNS switches execute entirely on your local machine via standard Windows Win32 and IP Helper APIs (`iphlpapi.dll`).
- **Third-Party DNS Providers:** When you switch to a DNS provider (such as Cloudflare `1.1.1.1` or AdGuard `94.140.14.14`), your computer's DNS queries are resolved by that respective provider subject to their independent privacy policies.

### 5. Disclaimer of Warranties ("AS IS")
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES, OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

### 6. Governing Law & Compliance
This Agreement and the use of the Software comply with applicable open source distribution regulations and consumer protection laws. You are responsible for ensuring that your use of custom DNS endpoints complies with your local telecommunication laws, organization policies, and network usage guidelines.

### 7. Your Rights as a User
- **Right to inspect:** The complete source code is published in the repository above. You may read, audit, or independently compile it at any time.
- **Right to modify and redistribute:** Granted under the MIT License, provided the original copyright notice is retained.
- **Right to use for free:** No payment, account, email address, or registration is required for any feature.
- **Right to data erasure:** Right-click the tray icon → **Clean uninstall** removes every trace in one step. Manually, deleting `%USERPROFILE%\.config\nodal\` and `%LOCALAPPDATA%\Nodal\`, and removing the `Nodal` registry Run value and scheduled task, achieves the same. See `PRIVACY_POLICY.md` §6.

### 8. Third-Party Components and Notices
Nodal is distributed as a single self-contained executable. It ships no third-party binaries, drivers, or bundled installers. It is compiled from the following open source Go modules, whose licenses are reproduced here for attribution:

| Component | Purpose | License |
| --- | --- | --- |
| Go standard library (including `syscall`) | Core runtime and Win32 FFI | BSD-3-Clause |
| `golang.org/x/sys/windows` | Windows API bindings and registry access | BSD-3-Clause |
| `github.com/BurntSushi/toml` | Parsing of `config.toml` | MIT |

Windows, Win32, WinUI, Cloudflare, AdGuard, Quad9, NextDNS and all other product names are trademarks of their respective owners and are used only for identification of interoperable services. Nodal is not affiliated with, endorsed by, or sponsored by any of these parties.

### 9. Changes to These Terms
Any revision of this document is published in the repository with an updated "Last Updated" date. Because Nodal collects no contact information, continued use of a newer release constitutes acceptance of the revised terms. If you do not accept a revision, uninstall the Software (see `docs/installation.md`).

### 10. Contact
Questions about licensing, terms, or consumer rights: open an issue at https://github.com/nodal/dns-switcher/issues.
