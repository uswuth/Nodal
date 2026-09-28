<p align="center">
  <img src="./dns-nodal-logo.svg" width="96" alt="Nodal logo" />
</p>

<h1 align="center">Nodal</h1>

<p align="center"><b>One-click DNS switching, from your system tray.</b><br/>No dashboard. No bloat. Just click and you're on Cloudflare, AdGuard, or DHCP.</p>

<p align="center">
  <img src="https://img.shields.io/badge/Windows-11-0F6CBD?style=flat-square&logo=windows11&logoColor=white" alt="Windows 11" />
  <img src="https://img.shields.io/badge/Go-1.21-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/tray-native Win32-e8792e?style=flat-square" alt="native tray" />
  <img src="https://img.shields.io/badge/idle-%3C5_MB_RAM-3BA0F2?style=flat-square" alt="lightweight" />
  <img src="https://img.shields.io/badge/switch-%3C1s-22c55e?style=flat-square" alt="fast switch" />
</p>

---

## Product identity

| | |
| --- | --- |
| **Publisher** | Nodal Open Source Project |
| **Description** | Native Windows 11 DNS Switcher — one-click DNS switching from the system tray |
| **Version** | 1.0.0 |
| **Price** | Free of charge — no subscription, trial, license key, or account |
| **License** | MIT |
| **Platform** | Windows 10 / 11, 64-bit |

## Install

Download the latest `Nodal-Setup-<version>.exe` from the
[Releases](https://github.com/nodal/dns-switcher/releases) page, verify its SHA-256 hash against
`SHA256SUMS.txt`, then run the wizard. It walks through the privacy notice, the license terms, the
destination folder and optional tasks, and registers the privileged DNS worker. A portable
`nodal.exe` is published alongside the installer.

Done with it? Right-click the tray icon → **Clean uninstall**. It restores automatic (DHCP) DNS on
every adapter, removes the scheduled task, the startup entry, your presets and the app itself, and
touches nothing else on the PC.

Full stage-by-stage instructions, unattended-install switches and uninstall details:
[Installation guide](docs/installation.md).

## Build

```powershell
.\build.ps1                 # bin\nodal.exe + dist\Nodal-Setup-<version>.exe
.\build.ps1 -SkipInstaller  # executable only
```

Requires Go 1.21 or newer, plus Inno Setup 6.3 or newer for the installer. The build embeds the
publisher name, product description and version from `cmd\nodal\versioninfo.json`.

## Why Nodal?

- **Fastest way to switch DNS on Windows** — one tray click, no Settings maze, no admin prompts every time.
- **Feels like Windows 11** — native flyout, dark/light + accent aware.
- **Stays out of the way** — <5 MB idle, zero background CPU, autostart optional.

## Defaults

`DHCP` · `Cloudflare 1.1.1.1` · `AdGuard` · `+ Custom` (up to 5)

> Need Quad9, NextDNS, Family filters? → [DNS Directory](docs/dns-guide.md)

## Config

`%USERPROFILE%\.config\nodal\config.toml`

```toml
autostart = false
privacy_mode = "visible"   # "visible" | "masked" | "hidden"

[[dns]]
name = "Cloudflare"
primary = "1.1.1.1"
secondary = "1.0.0.1"
```

## License & Policies

Nodal is **free of charge** and released under the [MIT License](LICENSE). It contains no telemetry,
makes no network requests of its own, and keeps all data on your machine.

- [Terms of Service & EULA](TERMS_AND_POLICY.md) — license, system rights Nodal uses, warranty disclaimer, third-party notices
- [Privacy Policy](PRIVACY_POLICY.md) — what stays local, what is never collected, how to erase everything
- [Installation guide](docs/installation.md) — download, verify, install, update, uninstall

