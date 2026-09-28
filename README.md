<div class="hero">
  <img class="hero-img" src="{{ '/dns-nodal-logo.svg' | relative_url }}" width="100" alt="Nodal" />
  <h1>Nodal</h1>
  <p class="subtitle">
    Your DNS, switched in one click — right from the tray.<br/>
    Native Windows. No dashboard. No account. No telemetry. Free forever.
  </p>
</div>

<div class="badge-row">
  <img src="https://img.shields.io/badge/Windows-11-0F6CBD?style=flat-square&logo=windows11&logoColor=white" alt="Windows 11" />
  <img src="https://img.shields.io/badge/Go-1.21-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/tray-native%20Win32-e8792e?style=flat-square" alt="native tray" />
  <img src="https://img.shields.io/badge/idle-%3C5_MB_RAM-3BA0F2?style=flat-square" alt="lightweight" />
  <img src="https://img.shields.io/badge/switch-%3C1s-22c55e?style=flat-square" alt="fast switch" />
</div>

---

## Preview

<div class="preview-grid">
  <div class="preview-card">
    <h3>Dark mode</h3>
    <picture>
      <source srcset="{{ '/docs/images/preview-dark-flyout.webp' | relative_url }}" type="image/webp">
      <img src="{{ '/docs/images/preview-dark-flyout.png' | relative_url }}" alt="Nodal flyout — dark mode" class="inline-img" />
    </picture>
  </div>
  <div class="preview-card">
    <h3>Light mode</h3>
    <picture>
      <source srcset="{{ '/docs/images/preview-light-flyout.webp' | relative_url }}" type="image/webp">
      <img src="{{ '/docs/images/preview-light-flyout.png' | relative_url }}" alt="Nodal flyout — light mode" class="inline-img" />
    </picture>
  </div>
</div>

## Why Nodal

<div class="feature-grid">
  <div class="feature-card">
    <h3>One click</h3>
    <p>Switch resolvers straight from the tray; Windows Settings never opens.</p>
  </div>
  <div class="feature-card">
    <h3>Native Win32</h3>
    <p>Dark/light and accent aware. No Electron, no web views, no runtimes.</p>
  </div>
  <div class="feature-card">
    <h3>Silent privilege</h3>
    <p>Approve elevation once; every later switch runs without prompts.</p>
  </div>
  <div class="feature-card">
    <h3>Zero footprint</h3>
    <p>No drivers, no services, no telemetry, no network requests of its own.</p>
  </div>
  <div class="feature-card">
    <h3>Clean uninstall</h3>
    <p>One tray action reverses every change Nodal ever made.</p>
  </div>
  <div class="feature-card">
    <h3>Featherweight</h3>
    <p>One self-contained executable, under 5 MB idle.</p>
  </div>
</div>

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

<table>
  <tbody>
    <tr><th>Publisher</th><td>Nodal Open Source Project</td></tr>
    <tr><th>Price</th><td>Free of charge — no subscription, no trial, no license key, no account</td></tr>
    <tr><th>License</th><td>MIT</td></tr>
    <tr><th>Platform</th><td>Windows 10 / 11, 64-bit</td></tr>
  </tbody>
</table>

## Policies

Nodal contains no telemetry and makes no network requests of its own — all data stays on your
machine.

- [Privacy Policy](PRIVACY_POLICY.md) — what stays local, what is never collected, how to erase everything
- [Terms of Service & EULA](TERMS_AND_POLICY.md) — system rights, warranty disclaimer, third-party notices
- [DNS Directory](docs/dns-guide.md) — curated resolvers ready to paste into a custom profile

---

<div class="site-footer">
  Built with [Go](https://go.dev/) · Distributed under the [MIT License](LICENSE) ·
  <a href="https://github.com/uswuth/Nodal">Source on GitHub</a>
</div>

