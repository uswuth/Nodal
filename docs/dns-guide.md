# DNS Directory

Pick a preset, paste it into Nodal via `+ Custom DNS`. That's it.

| Want | Preset | Primary · Secondary |
|------|--------|---------------------|
| Block ads | **AdGuard** | `94.140.14.14` · `94.140.15.15` |
| Block ads + adult | **AdGuard Family** | `94.140.14.15` · `94.140.15.16` |
| Fastest | **Cloudflare** | `1.1.1.1` · `1.0.0.1` |
| Malware shield | **Quad9** | `9.9.9.9` · `149.112.112.112` |
| Family filter | **Cloudflare Family** | `1.1.1.3` · `1.0.0.3` |
| Custom rules | **NextDNS** | `45.90.28.0` · `45.90.30.0` |

## Paste this

```toml
[[dns]]
name = "Quad9 Security"
primary = "9.9.9.9"
secondary = "149.112.112.112"
```

> Hotel / office Wi-Fi broken after switching? Go back to **DHCP**.

