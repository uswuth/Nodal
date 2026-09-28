<p align="center">
  <a href="https://github.com/uswuth/Nodal">
    <img src="dns-nodal-logo.svg" width="100" alt="Nodal" />
  </a>
</p>

<h1 align="center" style="font-size:2.5rem;font-weight:800;margin:0.5rem 0 0.25rem;color:#1C2833;letter-spacing:-0.02em;">Nodal</h1>

<p align="center" style="color:#6C757D;font-size:1.15rem;max-width:600px;margin:0 auto 1.5rem;line-height:1.6;">
  Your DNS, switched in one click — right from the tray.<br>
  Native Windows. No dashboard. No account. No telemetry. Free forever.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Windows-11-0F6CBD?style=flat-square&logo=windows11&logoColor=white" alt="Windows 11" />
  <img src="https://img.shields.io/badge/Go-1.21-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/tray-native%20Win32-e8792e?style=flat-square" alt="native tray" />
  <img src="https://img.shields.io/badge/idle-%3C5_MB_RAM-3BA0F2?style=flat-square" alt="lightweight" />
  <img src="https://img.shields.io/badge/switch-%3C1s-22c55e?style=flat-square" alt="fast switch" />
</p>

---

## Preview

<p align="center">
  <picture>
    <source srcset="docs/images/preview-dark-flyout.webp" type="image/webp">
    <img src="docs/images/preview-dark-flyout.png" alt="Nodal flyout — dark mode" style="max-width:100%;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,0.12);" />
  </picture>
</p>

<p align="center">
  <picture>
    <source srcset="docs/images/preview-light-flyout.webp" type="image/webp">
    <img src="docs/images/preview-light-flyout.png" alt="Nodal flyout — light mode" style="max-width:100%;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,0.12);" />
  </picture>
</p>

## Why Nodal

| | |
| --- | --- |
| **One click** | Switch resolvers straight from the tray; Windows Settings never opens. |
| **Native Win32** | Dark/light and accent aware. No Electron, no web views, no runtimes. |
| **Silent privilege** | Approve elevation once; every later switch runs without prompts. |
| **Zero footprint** | No drivers, no services, no telemetry, no network requests of its own. |
| **Clean uninstall** | One tray action reverses every change Nodal ever made. |
| **Featherweight** | One self-contained executable, under 5 MB idle. |

## Install

Download the installer from [Releases](https://github.com/uswuth/Nodal/releases) and run it —
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
| **License** | <a href="LICENSE">MIT</a> |
| **Platform** | Windows 10 / 11, 64-bit |

## Policies

Nodal contains no telemetry and makes no network requests of its own — all data stays on your
machine.

- [Privacy Policy](PRIVACY_POLICY.md) — what stays local, what is never collected, how to erase everything
- [Terms of Service & EULA](TERMS_AND_POLICY.md) — system rights, warranty disclaimer, third-party notices
- [DNS Directory](docs/dns-guide.md) — curated resolvers ready to paste into a custom profile

---

<p align="center" style="color:#6C757D;font-size:0.9rem;margin-top:2rem;border-top:1px solid #E9ECEF;padding-top:1rem;">
  Built with <a href="https://go.dev/">Go</a> · Distributed under the <a href="LICENSE">MIT License</a> ·
  <a href="https://github.com/uswuth/Nodal">Source on GitHub</a>
</p>

