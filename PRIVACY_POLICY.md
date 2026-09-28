# Privacy Policy & Consumer Safety Notice

**Effective Date:** September 2026  
**Product:** Nodal  
**Publisher:** Nodal Open Source Project  
**Contact / Repository:** https://github.com/uswuth/Nodal

---

## 1. Principles of Data Protection
Nodal is designed with the highest standards of data minimization and consumer privacy:
- **No Telemetry:** We do not include any tracking pixels, analytics SDKs, Google Analytics, telemetry beacons, or crash reporters.
- **No User Accounts:** You do not need to register, log in, or provide email addresses or payment information.
- **No Remote Servers:** The application has no external control servers, database backends, or cloud storage. It does not phone home.

## 2. Information Handled Locally
The application only reads and writes the following items stored on your physical device:
1. **`%USERPROFILE%\.config\nodal\config.toml`**: Contains your DNS profile names and resolver addresses — the built-in presets and your own custom entries.
2. **`%LOCALAPPDATA%\Nodal\pending.json` & `result.json`**: Temporary transient IPC files used to pass the selected profile to the background worker during DNS updates.
3. **Registry Run Key (`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`)**: Contains the path to `nodal.exe` solely if autostart is toggled on.
4. **Network adapter DNS configuration**: The primary/secondary resolver addresses of the adapter you switch. Selecting a profile writes them, selecting **DHCP** clears them again, and uninstalling Nodal restores the resolvers provided by your router automatically.

## 3. Consumer Safety & Network Integrity
- **Authenticity & Integrity:** Nodal source code is open for review. Anyone can verify and reproduce binary builds directly from the GitHub repository.
- **No Privilege Creep:** Nodal does not run permanently with administrative rights. The tray application runs with standard user rights. Only the atomic DNS switch action is dispatched through a restricted scheduled task worker.
- **Safe Uninstallation:** Nodal installs no drivers and leaves no hidden persistence hooks. Uninstalling restores Windows' automatic (DHCP-provided) resolvers on every network adapter, removes the privileged worker task, the optional startup entry, your settings, and finally the application itself. Right-click the tray icon and choose **Clean uninstall** to perform all of it in one step; the only objects removed are those Nodal created.

## 4. Children’s and General Consumer Rights
Nodal does not target, collect, or store information from any consumer or minor. It is completely safe, transparent, and private.

## 5. What Nodal Never Collects
For absolute clarity, the following categories of data are never collected, generated, or transmitted by the Software:
- Name, email address, phone number, postal address, or any other identifier.
- Browsing history, visited websites, DNS query logs, or resolver statistics.
- Device identifiers, hardware fingerprints, IP geolocation, or MAC addresses.
- Crash dumps, stack traces, usage counters, feature telemetry, or A/B experiment data.
- Payment, billing, or subscription information (the Software is free of charge).

There is no analytics endpoint, no update-check server, and no remote configuration channel compiled into the binary.

## 6. Data Retention and Complete Erasure
No data leaves your machine, so no retention period applies on our side. Everything can be erased locally, without contacting anyone:

**One-action erasure:** right-click the tray icon and choose **Clean uninstall** (the equivalent command is `nodal.exe --clean-uninstall`). It restores automatic (DHCP) DNS on every network adapter, removes the privileged worker task, the optional startup entry, both data folders listed below and the application itself — and touches nothing else on the PC.

Manual erasure, step by step:

1. Exit Nodal from the tray menu.
2. Delete the configuration folder: `%USERPROFILE%\.config\nodal\` (contains `config.toml` and `schema.json`).
3. Delete the runtime state folder: `%LOCALAPPDATA%\Nodal\` (contains `pending.json` and `result.json`).
4. Remove the scheduled task: `schtasks /delete /tn Nodal /f` from an **elevated** Command Prompt. The bundled uninstaller performs steps 1-4 and the registry cleanup below; it deletes the task automatically when it runs elevated, and otherwise prints the command to run.
5. Remove the startup entry, if enabled: delete the value `Nodal` under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`.

Every uninstall path also hands DNS back to Windows: `nodal.exe --reset-dns` clears the static resolvers on all adapters, so no override outlives the application. You can run that command on its own at any time to return to the resolvers your router assigns.

## 7. Changes to This Policy
Any change is published in the source repository with a new effective date. The current version is always the one bundled in the newest release.

## 8. Contact
Privacy or safety questions: open an issue at https://github.com/uswuth/Nodal/issues.
