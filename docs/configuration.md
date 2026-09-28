# Configuration

Nodal is controlled by a single local TOML document. It is read at launch and on every flyout open,
and is never transmitted anywhere.

## File locations

| Purpose | Path |
| --- | --- |
| Primary configuration | `%USERPROFILE%\.config\nodal\config.toml` |
| Fallback (when the profile directory is unavailable) | `%APPDATA%\nodal\config.toml` |
| Validation schema | `%USERPROFILE%\.config\nodal\schema.json` — regenerated on every launch |

Loading rules:

- Missing or empty file → built-in defaults are written and used.
- Malformed file → the last known-good configuration is retained for the session.
- The schema is JSON Schema draft-07 with `additionalProperties: false`, so editors validate the
  document against the exact key set while you type.

## Top-level keys

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| `autostart` | boolean | `false` | Mirrors launch-at-sign-in. When true, the `Nodal` value under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` points at the executable. |
| `privacy_mode` | enum: `visible` \| `masked` \| `hidden` | `visible` | How resolver addresses render in the flyout: full address shown / trailing octets masked / address line omitted (profile name only). |
| `theme` | enum: `auto` \| `dark` \| `light` | `auto` | Flyout and menu theme. `auto` follows the system Apps-mode setting. |
| `show_custom_button` | boolean | `true` | Shows or hides the custom-profile row at the bottom of the flyout. |
| `hide_ip` | boolean | — | Legacy key. When true and `privacy_mode` is unset it behaves as `masked`. Superseded by `privacy_mode`. |
| `dns` | array of profile objects | built-in presets | The profile list rendered in the flyout, in file order. 1–5 entries. |

## Profile keys (`dns[]`)

| Key | Type | Description |
| --- | --- | --- |
| `name` | string, required | Display label in the flyout and tray tooltip. |
| `tag` | string, optional | Short category badge rendered on the profile row. |
| `primary` | string, required | Primary IPv4 resolver in dotted-quad notation. Empty selects automatic (DHCP) resolution. |
| `secondary` | string, optional | Secondary IPv4 resolver, used as failover. |
| `primary_v6` | string, optional | Primary IPv6 resolver. |
| `secondary_v6` | string, optional | Secondary IPv6 resolver. |
| `dot` | string, optional | DNS-over-TLS endpoint hostname. Persisted with the profile; the switcher applies plain resolver addresses only. |
| `doh` | string, optional | DNS-over-HTTPS endpoint URL. Persisted with the profile; the switcher applies plain resolver addresses only. |

Constraints and behaviour:

- `primary` and `secondary` must match the dotted-quad pattern (octets 0–255) or be empty — enforced
  by `schema.json`.
- At most 5 profiles are kept; additional entries are rejected by the schema.
- A profile whose resolver fields are all empty applies *obtain DNS automatically* for both address
  families.
- Selecting a profile writes the resolver addresses through `SetInterfaceDnsSettings` and mirrors
  them in the Windows Settings UI; selecting an empty profile clears the static override.
- Switches are dispatched to the privileged scheduled-task worker when the tray process is not
  elevated; profiles never contain credentials.