# refract

[![en](https://img.shields.io/badge/lang-English-blue.svg)](https://github.com/quyxishi/refract/blob/main/README.md)
[![ru](https://img.shields.io/badge/lang-Russian-blue.svg)](https://github.com/quyxishi/refract/blob/main/README.ru.md)

Refract is a policy enforcement service that provides real-time session concurrency control for Xray-core.

It monitors access logs to detect concurrent connections from different IP addresses under the same user identity and enforces limits by interacting directly with the Linux kernel via Netlink and IPSet.

## Features

- **Real-time Enforcement**: Detects and blocks unauthorized concurrent sessions within milliseconds.
- **Kernel-Level Blocking**: Uses `ipset` and `iptables` to efficiently drop packets at the network layer.
- **Instant Session Termination**: Uses `SOCK_DIAG` (netlink) to immediately kill existing TCP connections of banned users.
- **Zero-Copy Log Parsing**: Highly efficient log tailing designed for high-throughput environments.
- **Docker Ready**: Fully supports `network_mode: host` with `NET_ADMIN` capabilities.

## Architecture

Refract operates as a sidecar to Xray-core. It tails the access log, tracks the state of active users, and when a violation is detected, it:

1. Adds the infringing IP to the `refract_banned_users` kernel ipset (with a configurable timeout).
2. Immediately terminates the specific TCP socket.
3. Ensures an `iptables` rule exists to drop traffic from that ipset.

## Usage

### Prerequisites

- Root privileges (or Docker in rootful mode with `CAP_NET_ADMIN`) are required to manage ipsets, iptables, and sockets.
- Xray configured to log real client IPs (via `proxy_protocol` or direct connection).
- Linux kernel with `ipset` and `netlink` support.

### Configuration

Refract is configured via command-line flags:

```shell
./refract \
    --proto=tcp \
    --dport=443 \
    --window=5s \
    --timeout=1m \
    --access.log=/var/log/xray/access.log \
    --block.log=/var/log/xray/block.log
```

| Flag         | Description                                                                                           | Default                  |
| ------------ | ----------------------------------------------------------------------------------------------------- | ------------------------ |
| `proto`      | The transport protocol (tcp/udp) used to filter connections for enforcement.                          | tcp                      |
| `dport`      | The destination port of the connections that should be monitored and blocked.                         | 443                      |
| `window`     | Time window for checking concurrency. Allows brief overlap for network switching (e.g., WiFi to LTE). | 5s                       |
| `timeout`    | Duration to enforce the ban on a conflicting IP before automatically lifting the restriction.         | 1m                       |
| `access.log` | Path to Xray access log.                                                                              | /var/log/xray/access.log |
| `block.log`  | Path to Refract audit log (trail of enforcement actions).                                             | /var/log/xray/block.log  |

### From Docker Compose

In order to run, Refract requires `host` network mode to see real client IPs and manage the host's firewall.

```yaml
services:
  refract:
    image: rxyvea/refract:latest
    container_name: refract
    restart: unless-stopped
    network_mode: host
    cap_add:
      - NET_ADMIN
    volumes:
      - /var/log/xray/access.log:/var/log/xray/access.log:ro
      - /var/log/xray/block.log:/var/log/xray/block.log
```

### From Binary

If you prefer running Refract as a systemd service without Docker, follow these steps.

1. Install required dependencies:

```bash
# For Debian/Ubuntu
sudo apt install -y ipset iptables iproute2
# For CentOS/RHEL/Fedora
sudo dnf install -y ipset iptables iproute
# For Alpine
apk add ipset iptables iproute2
```

2. Either clone repository and build (via `make build`) or pull latest release from [Releases page](https://github.com/quyxishi/refract/releases/latest).
3. Extract the binary to system directory and make it executable:

```shell
chmod +x /opt/refract/refract
```

4. Create a systemd service configuration file:

```shell
mkdir -p /etc/default
cat <<EOF > /etc/default/refract
REFRACT_PROTO="tcp"
REFRACT_DPORT="443"
REFRACT_WINDOW="5s"
REFRACT_TIMEOUT="1m"
REFRACT_ACCESS_LOG="/var/log/xray/access.log"
REFRACT_BLOCK_LOG="/var/log/xray/block.log"
EOF
```

5. Copy [systemd service file](/refract.service) to `/etc/systemd/system/refract.service` and start the service:

```shell
systemctl daemon-reload
systemctl enable --now refract
systemctl status refract
```

---

> [!WARNING]
> **If working behind a reverse proxy** (Nginx/HAProxy), make sure the real client IP reaches Xray via the PROXY protocol; otherwise, you may end up blocking localhost or your server IP instead of the actual offender.

### Reverse Proxy Configuration

###### Xray example configuration:

```json
{
  "log": {
    "error": "/var/log/xray/error.log",
    "access": "/var/log/xray/access.log",
    "loglevel": "warning"
  },
  "inbounds": [
    {
      "tag": "NIDX00-INBOUND-IDX00",
      "port": 14443,
      "listen": "127.0.0.1",
      "protocol": "vless",
      "sniffing": {
        "enabled": true,
        "destOverride": ["http", "tls", "quic"]
      },
      "streamSettings": {
        "network": "xhttp",
        "security": "reality",
        "sockopt": {
          "acceptProxyProtocol": true // Accept PROXY v1/v2 from the reverse proxy
        }
      }
    }
  ]
}
```

###### Nginx example configuration:

```nginx
stream {
    # Map for routing by SNI, upstreams, and etc.

    server {
        listen 443;
        listen [::]:443;

        ssl_preread on;

        # Send PROXY protocol to backend
        proxy_protocol on;
        # Xray inbound
        proxy_pass $backend_name;
    }
}
```

###### HAProxy example configuration:

```haproxy
backend xray_backend
    mode tcp
    server xray1 127.0.0.1:14443 send-proxy-v2
```

This ensures that Xray receives the real client IP address in it's access logs, allowing refract to restrict the correct IP addresses.

## Known Issues

### IPSet Timeout Persistence

Refract uses a kernel ipset (named `refract_banned_users`) to track banned IPs.
If you change the `--timeout` value after the service has already created this set, **new timeout may not apply immediately**.

This happens because `ipset create` does not overwrite an existing set's parameters. The kernel retains the timeout value defined when the set was first created.

**Solution:**
If you change the timeout configuration, you must manually destroy the existing set for the change to take effect:

```shell
# Find refract rule index
sudo iptables -L INPUT --line-numbers | grep refract

# Drop rule, replace $X with rule index
sudo iptables -D INPUT $X

# Flush and destroy the old set
sudo ipset destroy refract_banned_users
```

## Development

###### Build

```shell
make build
```

###### Running

```shell
make run
```

## License

MIT License, see [LICENSE](https://github.com/quyxishi/refract/blob/main/LICENSE).
