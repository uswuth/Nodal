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

