# sm-cli

sm-cli is a command-line tool written in Go for managing server operations via Redfish API. It provides commands to interact with BMC (Baseboard Management Controller) systems such as powering on/off servers, managing virtual media, and setting boot options.

I started using this tool because it takes a lot of time to log in to all IPMIs and adjust certain things, especially when you are dealing with 100+ servers. This tool allows you (with a simple bash statement) to walk over all hosts and run tasks in batches.

## Features
- Power Management: Control server power state (on, off, restart).
- Virtual Media: Insert and eject virtual media.
- Boot Options: Set boot order to CD or PXE once.

I am adding more features to this tool as I continue to work on it. Feel free to contribute or suggest new features!

## Installation
Clone the repository and build the binary:

```
git clone https://github.com/Charlie-Root/smcli.git
cd sm-cli
go build -o sm-cli main.go
```
Make sure to have Go installed on your system.

## Usage
The sm-cli tool supports the following commands:

### Power Management
```
./sm-cli power [on|off|restart] <hostname>
```

#### Examples:

```
# Power on a server
./sm-cli power on host1

# Power off a server
./sm-cli power off host2

# Restart a server
./sm-cli power restart host3
``` 

### Virtual Media Management
```
./sm-cli media [insert|eject|status] <hostname>
```

#### Examples:

```
# Insert virtual media from ISO image
./sm-cli media insert host1

# Eject virtual media
./sm-cli media eject host2

# Check virtual media status
./sm-cli media status host3
```

### Boot Options
```
./sm-cli boot [cd|pxe] <hostname>
```

#### Examples:

```
# Set boot order to CD once
./sm-cli boot cd host1

# Set boot order to PXE once
./sm-cli boot pxe host2
``` 

## Configuration

Configure your servers in inventory.yaml:

```yaml
hosts:
  - name: host1
    bmc_address: x.x.x.x
    username_password: ADMIN:xxxxxx
    iso_image: http://some.server/debian.iso
```
Ensure each server entry includes its hostname, BMC IP address, username/password for authentication, and ISO image URL for virtual media operations.

## Additional Notes

### SSL Verification: 
By default, SSL verification is skipped for BMC connections. Modify main.go to change this behavior if required.

### Verbose Mode: 
Use -v or --verbose flag to enable detailed output for requests.

# Contributing

Contributions are welcome! Fork the repository, create a new branch, make your changes, and submit a pull request.