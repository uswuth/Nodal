# Installing Nodal

**Product:** Nodal — Native Windows 11 DNS Switcher
**Publisher:** Nodal Open Source Project
**Description:** A lightweight system-tray utility that switches your adapter's DNS resolvers in one click.
**Price:** Free of charge — no subscription, no trial, no license key, no account.

Installation happens in explicit stages so you always know what is written to your PC. Nothing is
installed or changed silently, and no personal data leaves your machine.

---

## Stage 0 — Requirements

| Item | Requirement |
| --- | --- |
| Operating system | Windows 10 or Windows 11, 64-bit (x64, or Arm64 with x64 emulation) |
| Disk space | Under 10 MB |
| .NET / runtimes | None — the executable is self-contained |
| Administrator rights | Optional. A per-user install needs none; an all-users install and the one-time privileged worker registration do. |
| Network | Not required after installation. There is no update checker or licence server. |

## Stage 1 — Download from the repository

1. Open the source repository: <https://github.com/nodal/dns-switcher>
2. Go to **Releases** and pick the newest stable release.
3. Download either:
   - `Nodal-Setup-<version>.exe` — the guided installer (recommended), or
   - `nodal.exe` — the portable single file, no installation required.
4. Download `SHA256SUMS.txt` as well.

## Stage 2 — Verify what you downloaded

```powershell
Get-FileHash .\Nodal-Setup-1.0.0.exe -Algorithm SHA256
```

Compare the printed hash with the matching line in `SHA256SUMS.txt`. If the hashes differ, do not run
the file and re-download it from the official release page.

> The binaries are intentionally not code-signed (this is a zero-budget open source project), so
> Windows SmartScreen may show *"Windows protected your PC"*. Choose **More info → Run anyway** only
> after the hash check passes. You can also build the executable yourself from source (Stage 8).

## Stage 3 — Run the installer wizard

Each stage of the wizard is listed below together with exactly what it does.

| Wizard stage | What it shows / asks | What it changes |
| --- | --- | --- |
| 1. Install mode (only when not already elevated) | *Install for everyone* or *Install just for me*. Choosing everyone triggers the Windows UAC prompt. | Nothing |
| 2. Welcome | Publisher (**Nodal Open Source Project**), product description, and a "free of charge" statement | Nothing |
| 3. License agreement | `TERMS_AND_POLICY.md` — MIT licence, system rights Nodal uses (network adapter DNS settings, Task Scheduler, optional startup entry), warranty disclaimer | Nothing. You must accept to continue. |
| 4. Information | `PRIVACY_POLICY.md` — what is stored locally, what is never collected, how to erase everything | Nothing |
| 5. Select destination | Installation folder (default `%ProgramFiles%\Nodal`, or `%LocalAppData%\Programs\Nodal` for a per-user install) | Nothing yet |
| 6. Select tasks | Desktop shortcut (optional) and *Start Nodal automatically when I sign in to Windows* (optional) | Nothing yet |
| 7. Ready to install | Summary of the chosen folder and options | Nothing |
| 8. Installing | Copies files; registers the privileged DNS worker when elevated; stores your autostart choice | Files, scheduled task, `config.toml` |
| 9. Setup completed | Option to launch Nodal immediately | Starts the tray application |

The installer never installs drivers, kernel filters, browser extensions, VPN components, or
third-party binaries.

## Stage 4 — What gets installed

| Location | Content | Removed on uninstall |
| --- | --- | --- |
| `%ProgramFiles%\Nodal\` (or `%LocalAppData%\Programs\Nodal\`) | `nodal.exe`, `LICENSE`, `README.md`, `TERMS_AND_POLICY.md`, `PRIVACY_POLICY.md`, `docs\installation.md` | Yes |
| Start menu shortcuts | Nodal, Uninstall Nodal | Yes |
| Desktop shortcut | Only if you ticked the task | Yes |
| Scheduled task `Nodal` | Runs `nodal.exe --worker` with highest privileges so DNS switches need no repeated UAC prompts | Yes — silently by *Clean uninstall*, otherwise by the uninstaller when it runs elevated |
| `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` → `Nodal` | Only if you ticked the autostart task | Yes |
| `%USERPROFILE%\.config\nodal\config.toml` + `schema.json` | Your DNS presets and display preferences, created on first launch | Yes with *Clean uninstall*; otherwise only if you confirm at uninstall |
| `%LOCALAPPDATA%\Nodal\` | Transient `pending.json` / `result.json` hand-off files used during a DNS switch | Yes |
| Your adapter DNS resolvers | The primary/secondary addresses a Nodal preset wrote into the adapter configuration | Yes — every uninstall path restores automatic (DHCP) resolvers on all adapters |

## Stage 5 — First launch and the privileged worker

1. Nodal starts in the system tray (next to the clock). There is no window and no console.
2. `%USERPROFILE%\.config\nodal\config.toml` is created with the built-in presets if it does not
   exist yet — every key is documented in [Configuration](configuration.md).
3. Left-click the tray icon to open the flyout, then click a preset to switch DNS.
4. If the privileged worker task is missing, Nodal asks for elevation once via UAC and registers it.
   Approving it means later switches happen without a prompt. Declining is safe — Nodal simply asks
   again the next time you switch.

Applying DNS settings may briefly pause the active connection on some adapters. Selecting the
`DHCP` preset restores the automatically-assigned resolvers.

## Stage 6 — Unattended installation (administrators)

```powershell
# Per-machine install, no UI, autostart enabled, no restart prompt
Nodal-Setup-1.0.0.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /TASKS="startupicon"

# Per-user install with a desktop shortcut, logging the install to a file
Nodal-Setup-1.0.0.exe /SILENT /CURRENTUSER /TASKS="desktopicon" /LOG="%TEMP%\nodal-install.log"

# Uninstall silently — DNS returns to DHCP, the task, startup entry, settings and files are removed
"%ProgramFiles%\Nodal\unins000.exe" /VERYSILENT /SUPPRESSMSGBOXES
```

The software may be deployed through any standard software-distribution tool (Intune, Configuration
Manager, PDQ, Winget manifests). No licence key or activation step exists.

## Stage 7 — Updating and repairing

- **Update:** run a newer `Nodal-Setup-<version>.exe` over the existing installation. Your presets in
  `config.toml` are preserved.
- **Repair:** run the same installer again and choose the same destination.

Removal is covered at the end of this guide: [Uninstall](#uninstall).

## Stage 8 — Build and verify from source (optional)

Requires Go 1.21+ and, for the installer, Inno Setup 6.3+.

```powershell
git clone https://github.com/nodal/dns-switcher
cd dns-switcher

# Publisher metadata (goversioninfo + cmd\nodal\versioninfo.json -> resource.syso)
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
go generate ./cmd/nodal

# Executable (GUI subsystem, stripped)
go build -trimpath -ldflags "-H windowsgui -s -w" -o bin\nodal.exe ./cmd/nodal

# Installer -> dist\Nodal-Setup-<version>.exe
& "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe" /DMyAppVersion=1.0.0 installer\setup.iss
```

`go generate ./cmd/nodal` embeds the publisher name, product description and copyright into
`nodal.exe`. Releases are produced by CI (`.github/workflows/release.yml`) with the same commands.

Confirm the embedded publisher metadata after any build:

```powershell
(Get-Item .\bin\nodal.exe).VersionInfo | Format-List CompanyName, FileDescription, ProductName, LegalCopyright, FileVersion
```

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| "Windows protected your PC" (SmartScreen) | Expected for unsigned builds — verify the SHA-256 hash, then *More info → Run anyway*. |
| Installer refuses to start | Requires Windows 10/11 x64. `MinVersion=10.0` blocks older systems. |
| "Another version of this product is already installed" | Use *Settings → Apps* to remove the old entry, then re-run the installer. |
| No tray icon after launching | The icon may be in the hidden-icons overflow area of the taskbar. Launching Nodal twice exits immediately by design (single-instance guard). |
| DNS switch fails with an elevation error | Approve the UAC prompt so the privileged worker task can be registered, or reinstall with the all-users option and accept the elevation prompt. |
| Publisher missing in file properties | The build ran without `goversioninfo`. Run `go generate ./cmd/nodal` before `go build` (see Stage 8). |
| Antivirus flags the executable | Uncommon but reported for unsigned Go binaries. The source is public; build it yourself (Stage 8) and compare the hash, or add an exclusion. |

## Cost, rights and data protection

- Nodal is **free of charge** and licensed under the **MIT License** (`LICENSE`).
- No telemetry, analytics, accounts, or remote servers are involved. See `PRIVACY_POLICY.md`.
- The system capabilities Nodal uses are enumerated in `TERMS_AND_POLICY.md` §3, and the complete
  erasure steps are in `PRIVACY_POLICY.md` §6.

## Uninstall

Three ways out. All of them hand DNS back to Windows before anything is deleted.

### Clean uninstall (recommended)

Right-click the tray icon → **Clean uninstall**. Nodal lists exactly what it will do, asks for
confirmation, and then performs every step in order:

| Step | Action |
| --- | --- |
| 1 | Restores the automatic (DHCP) resolvers on **all** network adapters |
| 2 | Removes the `Nodal` startup entry from `HKCU\...\CurrentVersion\Run` |
| 3 | Removes the privileged worker scheduled task `Nodal` |
| 4 | Deletes `%LOCALAPPDATA%\Nodal\` and `%USERPROFILE%\.config\nodal\` |
| 5 | Runs the bundled uninstaller, which removes the files, shortcuts and the *Installed apps* entry |

Only objects created by Nodal are touched: other applications, drivers, registry keys and the
Windows firewall configuration are left untouched. Windows asks for administrator approval through
the standard UAC dialog when it is needed; declining it cancels the whole operation and changes
nothing.

### From Windows

*Settings → Apps → Installed apps → Nodal → Uninstall*, or the Start menu entry *Uninstall Nodal*.
This path restores DHCP DNS and removes the scheduled task and the startup entry as well; it asks
whether to delete your configuration and DNS presets.

### Portable copy

`nodal.exe` was never installed, so *Clean uninstall* removes the DNS overrides, the task, the
startup entry and the settings, and then tells you which file to delete. No uninstall entry exists
in *Installed apps* to remove.

### Command line

- `nodal.exe --clean-uninstall` — reverses every change; add `--yes` to skip the confirmation prompt.
- `nodal.exe --reset-dns` — restores automatic DNS only, keeps the application.
- `unins000.exe /VERYSILENT /SUPPRESSMSGBOXES` — unattended removal (see Stage 6). The uninstaller's
  questions are suppressed and answered with the clean-removal default (yes), so it still restores
  automatic (DHCP) DNS on every adapter and removes the scheduled task, the startup entry and the
  configuration folder — message boxes are never left waiting for a human.

### When the uninstall does not finish

| Symptom | Cause / fix |
| --- | --- |
| DNS still points at a custom provider after uninstall | The reset needs administrator rights; approve the UAC prompt. Re-run `nodal.exe --reset-dns` from an elevated prompt, or select *Obtain DNS server address automatically* in the adapter properties. |
| *Clean uninstall* reported nothing and the app is still running | The UAC prompt was declined, so nothing was changed. Run *Clean uninstall* again and approve the prompt. |
| Task `Nodal` survives a per-user uninstall | A per-user uninstall is not elevated, so the uninstaller prints the manual command. Use *Clean uninstall* instead, or run `schtasks /delete /tn Nodal /f` from an elevated prompt. |

