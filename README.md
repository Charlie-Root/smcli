# sm-cli

sm-cli is a command-line tool written in Go for managing Supermicro servers through their BMC (Baseboard Management Controller) using the Redfish API: power, boot overrides, virtual media, users, IP access control, and read-only diagnostics (system info, sensors, event log).

I started using this tool because it takes a lot of time to log in to all IPMIs and adjust certain things, especially when you are dealing with 100+ servers. sm-cli works on many hosts at once, in parallel.

## Features
- Power: status, on, off, restart, graceful shutdown.
- Diagnostics: system info (model, serial, BIOS/BMC firmware, health), sensors (temperatures, fans, PSUs, voltages), hardware event log.
- Virtual media: insert, eject, status.
- Boot: CD or PXE once (UEFI).
- Users: add an Administrator user.
- IP access control: enable, add rules.
- Hosts from an inventory file or from NetBox; select by name, rack, site, tag or role.
- Parallel execution, per-host results, text or JSON output, non-zero exit code when a host fails.
- Safety: `--dry-run` for changes, `--yes` required to change more than one host, passwords masked in verbose output.

## Installation
Grab a pre-built binary from the [releases](https://github.com/Charlie-Root/smcli/releases) page, or build it:

```
git clone https://github.com/Charlie-Root/smcli.git
cd smcli
go build -o sm-cli .
```

## Hosts

### Inventory file (default)
Configure your servers in `inventory.yaml` (or pass `--inventory <file>`, `-` reads it from stdin):

```yaml
hosts:
  - name: host1
    bmc_address: x.x.x.x
    username_password: ADMIN:xxxxxx
    iso_image: http://some.server/debian.iso
```

### NetBox
With `--source netbox` (or `SMCLI_SOURCE=netbox`), sm-cli looks hosts up in NetBox instead. Set `NETBOX_URL` and `NETBOX_TOKEN`. For each device it follows:

device → `oob_ip` → the interface that IP is assigned to → interface custom fields `ipmi_username` and `ipmi_password`.

Credentials are only held in memory. Devices without an OOB IP or without credentials are reported as failed for that host.

Select hosts by name, or with `--rack`, `--site`, `--tag` and `--role` (NetBox slugs; filters combine):

```
sm-cli --source netbox power status --rack A.R10 --role server
```

## Usage

Hosts are given after the action. Every command accepts one or more hosts.

### Diagnostics (read-only)
```
sm-cli power status host1 host2
sm-cli info host1
sm-cli sensors host1 [--problems]
sm-cli sel host1 [--since 24h] [--limit 50] [--manager]
sm-cli media status host1
```
`sel --since 0` shows the whole log; `--manager` adds the BMC's own logs.

### Power
```
sm-cli power [on|off|restart|shutdown] <host...>
```
`off` and `restart` are forced; `shutdown` asks the OS to shut down (ACPI).

### Boot
```
sm-cli boot [cd|pxe] <host...>
```

### Virtual media
```
sm-cli media [insert|eject|status] <host...> [--iso http://some.server/debian.iso]
```

### Users and access control
```
sm-cli user <username> <password|-> <host...>
sm-cli acl enable <host...>
sm-cli acl add <address> <prefix> <policy> <host...>
```
Use `-` as password to read it from stdin, so it does not show up in the process list or shell history.

### Global flags
| Flag | Default | Meaning |
|---|---|---|
| `--source` | `file` (`SMCLI_SOURCE`) | `file` or `netbox` |
| `--inventory` | `inventory.yaml` | inventory file, `-` for stdin |
| `--rack`, `--site`, `--tag`, `--role` | | NetBox selection |
| `--parallel` | `5` | hosts at the same time |
| `--timeout` | `15s` | per request |
| `-o`, `--output` | `text` | `text` or `json` |
| `--dry-run` | | show changes without sending them |
| `-y`, `--yes` | | required to change more than one host |
| `-v`, `--verbose` | | log requests to stderr, passwords masked |

Exit code is 0 when every host succeeded, 1 otherwise.

## Notes

### TLS
Certificate verification is skipped for BMC connections (BMCs use self-signed certificates). NetBox connections are verified.

### Compatibility
The command syntax of earlier versions still works (`sm-cli power on host1`, `sm-cli user <name> <password> <host>`, `sm-cli acl add ...`).

## Contributing

Contributions are welcome! Fork the repository, create a new branch, make your changes, run `go test ./...`, and submit a pull request.

## References
https://www.supermicro.com/manuals/other/redfish-ref-guide-html/Content/general-content/available-apis.htm
