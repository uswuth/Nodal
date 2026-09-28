<p align="center">
  <img src="./dns-nodal-logo.svg" width="96" alt="Nodal logo" />
</p>

<h1 align="center">Nodal</h1>

<p align="center"><b>Your DNS, switched in one click — right from the tray.</b><br/>Native Windows. No dashboard. No account. No telemetry. Free forever.</p>

<p align="center">
  <img src="https://img.shields.io/badge/Windows-11-0F6CBD?style=flat-square&logo=windows11&logoColor=white" alt="Windows 11" />
  <img src="https://img.shields.io/badge/Go-1.21-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/tray-native Win32-e8792e?style=flat-square" alt="native tray" />
  <img src="https://img.shields.io/badge/idle-%3C5_MB_RAM-3BA0F2?style=flat-square" alt="lightweight" />
  <img src="https://img.shields.io/badge/switch-%3C1s-22c55e?style=flat-square" alt="fast switch" />
</p>

---

## Preview

**Dark mode** — tray menu · flyout

<p align="center">
  <img src="docs/images/preview-dark-menu.png" width="49%" alt="Nodal tray menu — dark mode" />
  <img src="docs/images/preview-dark-flyout.png" width="49%" alt="Nodal flyout — dark mode" />
</p>

**Light mode** — tray menu · flyout

<p align="center">
  <img src="docs/images/preview-light-menu.png" width="49%" alt="Nodal tray menu — light mode" />
  <img src="docs/images/preview-light-flyout.png" width="49%" alt="Nodal flyout — light mode" />
</p>

## Why Nodal

- **One click** — switch resolvers straight from the tray; Windows Settings never opens.
- **Native Win32** — dark/light and accent aware. No Electron, no web views, no runtimes.
- **Silent privilege** — approve elevation once; every later switch runs without prompts.
- **Zero footprint** — no drivers, no services, no telemetry, no network requests of its own.
- **Clean uninstall** — one tray action reverses every change Nodal ever made.
- **Featherweight** — one self-contained executable, under 5 MB idle.

## Install

Download the installer from [Releases](https://github.com/nodal/dns-switcher/releases) and run it —
a portable build ships alongside it. Every stage, checksum verification and unattended switch:
[Installation guide](docs/installation.md).

## Uninstall

Right-click the tray icon → **Clean uninstall**. Automatic DNS is restored on every adapter, the
scheduled task, startup entry, presets and the app itself are removed — nothing else on the PC is
touched. Details: [Uninstall](docs/installation.md#uninstall).

## Configuration

Presets, theme and privacy behaviour are driven by one local file — every key, type and constraint
documented: [Configuration](docs/configuration.md).

## Product identity

| | |
| --- | --- |
| **Publisher** | Nodal Open Source Project |
| **Price** | Free of charge — no subscription, no trial, no license key, no account |
| **License** | MIT |
| **Platform** | Windows 10 / 11, 64-bit |

## Policies

Nodal contains no telemetry and makes no network requests of its own — all data stays on your
machine.

- [Privacy Policy](PRIVACY_POLICY.md) — what stays local, what is never collected, how to erase everything
- [Terms of Service & EULA](TERMS_AND_POLICY.md) — system rights, warranty disclaimer, third-party notices
- [DNS Directory](docs/dns-guide.md) — curated resolvers ready to paste into a custom profile

