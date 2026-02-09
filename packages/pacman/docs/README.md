# PACman - Proxy Auto Configuration Manager

PACman is a proxy management tool designed to simplify proxy and VPN management. It runs a rule-based proxy server on localhost (default: `127.0.0.1:11078`) and serves a Proxy PAC file for optimized browser traffic routing.

## How It Works

PACman runs a local proxy server on your machine (default: `127.0.0.1:11078`). It intercepts outgoing traffic from browsers or apps and routes it based on rules in a configuration file. If a request matches a rule, PACman tries the listed proxies in order until one succeeds. If no rule matches, traffic goes directly without a proxy.

PACman also manages VPN connections, handling setup and routing in the background.

### Supported Protocols

PACman supports the following proxy types:
- Cisco AnyConnect
- Palo Alto Networks GlobalProtect
- SSH Proxy
- SOCKS5 Proxy
- HTTP/HTTPS Proxy

### DNS Resolution

PACman respects system DNS settings. It checks `/etc/hosts` for hostname mappings first. If a match is found, it uses that IP for proxy connection. If no match exists and no rule applies to the hostname, PACman resolves the hostname via the proxy and checks if the resulting IP matches any rule (e.g., CIDR range) before routing.


### Proxy Chaining and Circular Reference Caution

PACman supports proxy chaining (e.g., routing an SSH tunnel through a VPN). However, avoid circular references (e.g., Proxy A to Proxy B back to A) as PACman does not detect loops, which may cause crashes or failures. Verify rules to prevent issues.

## Rule File Configuration

PACman uses a YAML (or JSON) configuration file to define proxies and routing rules. Below is the structure of the rule file.

### Config File Format

PACman uses a YAML (or JSON) file, typically at `~/.config/pacman/config`, to define listening settings, proxies, and routing rules. Settings are described using dot notation (e.g., `option.<name>.field`) to indicate structure.

- **`listen`**: Address and port for incoming connections.
  - **Format**: `host:port` (e.g., `127.0.0.1:11078`).
  - **Default**: `127.0.0.1:11078`.

- **`proxies.<name>`**: Proxy definitions, where `<name>` is a unique label (e.g., `proxies.cisco_vpn`) used in rules.
  - **`proxies.<name>.username`**: Username for authentication, if needed (e.g., `user`).
  - **`proxies.<name>.password`**: Password for authentication, if needed (e.g., `pass`).
  - **`proxies.<name>.protocol`**: Proxy type. Supported values:
    - `socks5`, `socks5h`: SOCKS5 proxy with or without hostname resolution.
    - `http`, `https`: Standard HTTP or HTTPS proxy.
    - `anyconnect`: Cisco AnyConnect VPN.
    - `gp`: Palo Alto Networks GlobalProtect VPN.
    - `ssh`: SSH-based proxy.
  - **`proxies.<name>.host`**: Hostname or IP, optionally with port (e.g., `proxy.example.com:1080`).
  - **`proxies.<name>.path`**: Optional path, often a usergroup for VPNs (e.g., `usergroup`).
  - **`proxies.<name>.options`**: Key-value pairs for additional settings.
    - **Global Option (All Protocols)**:
      - **`proxies.<name>.options.timeout`**: Idle timeout in seconds. Default: 3600 (1 hour). Use `0` to disable.
    - **Cisco AnyConnect (`anyconnect`)**:
      - **`proxies.<name>.options.token`**: Set to `totp` to prompt for a YubiKey TOTP token, appended to password.
    - **Palo Alto Networks GlobalProtect (`gp`)**:
      - **`proxies.<name>.options.token`**: Set to `totp` to prompt for a YubiKey TOTP token, appended to password.
    - **SSH Proxy (`ssh`)**:
      - **`proxies.<name>.options.identity`**: Path to private key file (e.g., `/path/to/privatekey`). Passphrase-protected files and local SSH agent not supported.

- **`rules.[]`**: Routing rules, where `[]` is the list position (e.g., `rules[0]`).
  - **`rules.[].hosts`**: Patterns to match hostnames or IPs. Traffic matching follows this rule.
    - **Domain Pattern Matching**: Matches hostnames using specific patterns. The table below shows pattern types and their behavior with example domains (`example.com` and `www.example.com`):
  
      | Match Type              | Pattern                  | example.com | www.example.com |
      |-------------------------|--------------------------|-------------|-----------------|
      | Zone/Wildcard Match     | `*.example.com`          | No          | Yes             |
      | Exact Match             | `example.com`            | Yes         | No              |
      | Leading Dot Notation    | `.example.com`           | Yes         | Yes             |
  
      Matching prioritizes exact matches (e.g., `example.com`) or specific subdomains (e.g., `sub.example.com`) over wildcards. Longer wildcard patterns (e.g., `*.sub.example.com`) are more specific than shorter ones (e.g., `*.example.com`).

    - **CIDR Notation**: Matches IP ranges for network addresses (e.g., `192.168.1.0/24`). Matching prioritizes narrower ranges (e.g., `192.168.1.0/26`) over broader ones (e.g., `192.168.1.0/24`).
  
    PACman uses a "most-specific-match-first" approach for rule selection, with config order as a tiebreaker. This prioritizes detailed rules for precise control.

  - **`rules.[].proxies`**: List of proxy labels (from `proxies.<name>`) to try in order until connection succeeds.
    - **Note**: Empty list skips proxying for matched hosts, useful for exclusions.

### Example Configuration

Example YAML configuration for PACman with proxies and rules.

```yaml
listen: 127.0.0.1:11078

proxies:
  cisco_vpn:
    username: user
    password: pass
    protocol: anyconnect
    host: vpn.example.com
    path: usergroup
    options:
      token: totp
      timeout: 0
  global_protect:
    username: admin
    password: secret
    protocol: gp
    host: gateway.example.com
    path: employee-group
    options:
      token: totp
      timeout: 7200
  socks_proxy:
    username: proxyuser
    password: proxypass
    protocol: socks5
    host: proxy.example.com:1080
    options:
      timeout: 1800
  ssh_tunnel:
    username: sshuser
    password: sshpass
    protocol: ssh
    host: proxy.example.com:22
    options:
      identity: /path/to/privatekey
      timeout: 0

rules:
  - hosts:
      - "*.internal.example.com"
      - ".example.com"
      - "192.168.1.0/24"
    proxies:
      - cisco_vpn
      - socks_proxy
  - hosts:
      - "specific.example.com"
    proxies:
      - global_protect
      - ssh_tunnel
```

Customize proxy names, host patterns, and order as needed.


## How to Use

Instructions for setting up PACman with applications and scenarios.

### Enabling the Proxy

Default addresses (configurable via `listen` in config):
- **HTTP Proxy**: `http://127.0.0.1:11078`
- **SOCKS5 Proxy**: `socks5://127.0.0.1:11078`

Point apps to HTTP or SOCKS5 addresses, or use the PAC file for automatic routing (see below).

### Browser Configuration on macOS

Two options for browser proxy routing on macOS:

1. **PAC File (Recommended for Performance)**:
   - In browser proxy settings, set "Automatic Proxy Configuration" URL to `http://127.0.0.1:11078/proxy.pac` (or custom address if `listen` changed).
   - Routes only matching traffic through PACman; non-matching traffic goes direct.

2. **SOCKS5 Proxy Directly**:
   - Set manual SOCKS5 proxy to host `127.0.0.1`, port `11078` (or custom address if `listen` changed).
   - Sends all traffic through PACman, potentially slower for non-proxied traffic.

For system-wide setup:
- Go to **System Settings > Network > [Active Network] > Details > Proxies**.
- Enable "Automatic Proxy Configuration" with `http://127.0.0.1:11078/proxy.pac`, or set SOCKS5 proxy to `127.0.0.1:11078`.

### Proxy PAC File Support

PACman serves a Proxy Auto-Config (PAC) file at `http://127.0.0.1:11078/proxy.pac` (or custom address if `listen` changed). This file routes only matching traffic through PACman, improving performance. Set this URL in browser or app proxy settings for automatic configuration.

### SSH Integration

Configure SSH to route traffic through PACman for matching hosts by adding to `~/.ssh/config`:

```ssh
Match exec "'/Applications/PACman.app/Contents/MacOS/pacman' check '%h'"
  ProxyJump 127.0.0.1:11078
```

SSH checks if target host (`%h`) matches `rules.[].hosts` via `pacman check`. If matched, traffic routes through PACman at `127.0.0.1:11078` (or custom address if `listen` changed) using a jump server.

### Terminal and Other HTTP-Based Applications

For tools supporting HTTP proxies (e.g., `curl`, `wget`, Rancher Desktop):
- Add to shell config (e.g., `~/.zshrc`, `~/.bashrc`):
  ```bash
  export HTTP_PROXY=http://127.0.0.1:11078  # or custom address if 'listen' changed
  export HTTPS_PROXY=http://127.0.0.1:11078 # or custom address if 'listen' changed
  export NO_PROXY=localhost,127.0.0.1
  ```
- For apps with explicit proxy settings (e.g., Rancher Desktop):
  - Set HTTP/HTTPS proxy to `http://127.0.0.1:11078` (or custom address).
  - Add `localhost`, `127.0.0.1` to "bypass proxy" list to avoid local traffic issues.

For SOCKS5-only apps, use `socks5://127.0.0.1:11078` (or custom address).
