# PACman - Proxy Auto Configuration Manager

PACman helps your browser, SSH, and other applications reach private networks through the right proxy or VPN, when they need it.

## Overview

Think of PACman as a local connection desk: you tell it which destinations need a special connection, and your applications ask it to connect on their behalf.

Unlike a traditional VPN client that routes all of your Mac's network traffic through a VPN, PACman runs an **opt-in proxy server** on your Mac, normally at `127.0.0.1:11078`. Only applications configured to use PACman send traffic to it. Other applications keep using their normal connections.

For example, you can send your company's internal sites through a work VPN while public websites continue using your regular internet connection. Starting PACman does not connect every configured VPN: connections start on demand.

### Routing Your Applications

PACman's configuration pairs destination rules with named connections. A rule can match a hostname, a domain and its subdomains, or an IP range. PACman chooses the most specific matching rule and tries that rule's proxies in order. Traffic sent to PACman without a matching rule connects directly.

There are three ways to put this to use:

- **Web browsers:** use PACman's Proxy Auto-Configuration (PAC) file at `http://127.0.0.1:11078/proxy.pac`. This is a small set of instructions that tells a compatible browser which destinations to send to PACman and which to reach directly. See [Browser Configuration on macOS](browser-configuration-on-macos.html).
- **SSH:** add the [SSH Integration snippet](ssh-integration.html) to your SSH configuration. It checks PACman's destination rules before choosing whether to connect through PACman, giving SSH a similar automatic choice.
- **Other applications:** configure their HTTP or SOCKS5 proxy settings to use PACman. PACman then applies its rules to the connections they send. See [Terminal and Other HTTP-Based Applications](terminal-and-other-http-based-applications.html).

The PAC file helps your browser decide whether to use PACman; PACman's rules decide which connection to use after traffic arrives. You maintain the rules in one place.

### Connections Start on Demand

Configured proxies begin **Offline**. This is normal: they are ready to be used, but PACman has not started them yet.

When an application requests a destination that needs a proxy or VPN, PACman starts that connection. The first request may take longer while the connection is established or you complete authentication. Later requests reuse the connection.

Once a connection is no longer in use, PACman can disconnect it after its idle timeout. The default is one hour; you can change or disable this for each proxy with [options.timeout](proxies.html#timeout). A later request starts it again. You can also choose **Connect** from a proxy's menu to start it yourself.

### Using the Menu Bar

Click the PACman menu-bar icon to see the local server address and your configured proxies. Each proxy shows its own status and offers the action that applies at that moment:

| Status | Meaning | Available action |
|--------|---------|------------------|
| ⚪ Offline | Waiting until needed. | **Connect** to start now. |
| 🟡 Connecting | Setting up the connection; authentication may need your attention. | **Cancel** to stop setup. |
| 🟢 Online | The connection is ready for use. | **Disconnect** to close it. |
| 🔴 Failed | Connection setup failed and automatic attempts are paused. | **Reset** to allow another attempt. |

The menu-bar icon animates while a connection is being established. A failure changes it to a ghost, so you can see that a proxy needs attention. Open the menu to find the affected proxy; once all failures are reset, the icon returns to its normal appearance or resumes the connection animation.

Under **Settings**, choose **Edit** to open your configuration, then **Reload** to apply your changes. **Help** opens this guide.

### Notifications and YubiKey OTP

**Allow PACman notifications in macOS** so you can see authentication requests, connection updates, and errors. PACman asks for notification permission when it starts. If you previously declined, enable PACman in **System Settings > Notifications**. If an expected prompt does not appear, check Notification Center and any Focus settings that may be silencing it.

For a VPN configured with [options.token: otp](proxies.html#token), PACman asks for a YubiKey one-time password during password authentication:

1. Wait for the **Authentication Required** notification for your proxy.
2. Choose **Reply**, or click the notification to open the input field in that proxy's menu. You can also open the PACman menu and the proxy's submenu yourself while authentication is waiting.
3. Focus the OTP input field, then touch your YubiKey's OTP sensor to enter the code. Make sure the field is focused before touching the key, so the code goes into the authentication prompt.
4. Submit the reply, or press **Return** in the menu's input field if your key has not already submitted it.

PACman appends the entered OTP to your configured password. This flow uses the YubiKey's OTP typing feature; it does not read a code from the key automatically. Notification permission enables the notification prompt and reply, while the menu input provides another way to respond if you miss the notification.

### When a Connection Fails

If a proxy or VPN fails to start, PACman marks it **Failed** and remembers the error. New requests do not keep retrying that failed setup. This helps avoid repeated authentication attempts that could lock your account.

Read the error notification and address the cause, such as an incorrect password, a missed OTP prompt, or an unreachable VPN server. Then open that proxy's menu and choose **Reset**, or use **Reset** on its error notification.

Reset clears the saved failure and returns the proxy to **Offline**. The next request that needs it can start a fresh connection; choose **Connect** if you want to try immediately. If a rule lists other proxies, PACman can still try those alternatives in order.

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

PACman reads a YAML or JSON file, normally at `~/.config/pacman/config`. The top level is an object with three settings: `listen`, `proxies`, and `rules`.

```yaml
listen: 127.0.0.1:11078
proxies:
  office:
    protocol: socks5
    host: proxy.example.com:1080
rules:
  - hosts: [".example.com"]
    proxies: [office]
```

`proxies` maps names you choose to proxy definitions. `rules` is a list of objects that match destinations and select proxy names. A destination without a matching rule connects directly.

The sections below use dot notation: `proxies.<name>.host` means the `host` field inside a named proxy, and `rules[].hosts` means the `hosts` field in each rule. Each setting includes its type, default or requirement, behavior, and an example. Defaults distinguish values supplied by PACman from values written into the initial configuration file.

### listen

**Type:** string in `host:port` format.

**Default:** `127.0.0.1:11078` in the initial configuration file; no implicit default when omitted.

Address for PACman's local HTTP and SOCKS5 proxy server, PAC file, and diagnostics. Required when PACman opens its own listener. With launchd socket activation, the listener comes from launchd instead.

**Example:**

```yaml
listen: 127.0.0.1:11078
```

### Proxies

**Type:** object mapping names to proxy definitions. Each definition can be an object or a URL string.

**Default:** no proxy definitions.

Choose a unique name for each proxy, then refer to that name in `rules[].proxies`. The fields of each `proxies.<name>` object are described below. URL strings use the equivalent form `protocol://username:password@host/path?option=value`; percent-encode special characters in URL credentials and query values.

**Example:**

```yaml
proxies:
  office:
    protocol: socks5
    host: proxy.example.com:1080
  backup: "socks5://proxy-backup.example.com:1080?timeout=1800"
```

#### proxies.<name>.protocol

**Type:** string.

**Required:** for each proxy; no default.

Selects the connection protocol. The built-in dialers accept:

| Value | Connection |
|-------|------------|
| `socks5`, `socks5h` | SOCKS5 proxy; both use the same dialer and can pass hostnames to the proxy. |
| `anyconnect` | Cisco AnyConnect VPN through OpenConnect. |
| `gp` | Palo Alto Networks GlobalProtect VPN through OpenConnect. |
| `ssh` | SSH connection used to forward traffic. |

**Example:**

```yaml
protocol: anyconnect
```

#### proxies.<name>.host

**Type:** string containing a hostname or IP address, optionally with a port.

**Required:** for each proxy; no default hostname.

The proxy or VPN server to connect to. SOCKS5 uses port `1080` when omitted; SSH uses `22`. VPN server addresses are passed to OpenConnect. Use brackets around IPv6 addresses when including a port.

**Example:**

```yaml
host: proxy.example.com:1080
```

#### proxies.<name>.username

**Type:** string.

**Default:** no username.

Username used to authenticate to the proxy or VPN server. Set it when the server requires username authentication.

**Example:**

```yaml
username: alice
```

#### proxies.<name>.password

**Type:** string.

**Default:** no password.

Password used to authenticate to the proxy or VPN server. For VPNs with `options.token: otp`, PACman appends the entered token to this password during authentication.

**Example:**

```yaml
password: "example-password"
```

#### proxies.<name>.path

**Type:** string.

**Default:** `/` for an object-form proxy definition.

Path appended to the VPN server address, such as an authentication usergroup. PACman adds a leading slash and cleans the path. Set it only when the VPN server requires a particular path.

**Example:**

```yaml
path: employee-group
```

#### proxies.<name>.options

**Type:** object mapping option names to strings.

**Default:** no additional options.

Options are converted to URL query parameters. Quote numeric values in YAML so their string type is explicit; JSON option values must also be strings. The settings below describe the options PACman uses.

**Example:**

```yaml
options:
  timeout: "1800"
```

<a id="timeout"></a>

##### proxies.<name>.options.timeout

**Type:** string containing an integer number of seconds.

**Default:** `"3600"` (one hour).

**Applies to:** all proxy protocols.

Closes an idle proxy connection after this interval. Use `"0"` to disable idle disconnection. This controls the shared proxy's idle lifetime, rather than the time allowed to establish a connection. An empty or non-integer value leaves the one-hour default in effect.

**Example:**

```yaml
options:
  timeout: "7200"
```

<a id="token"></a>

##### proxies.<name>.options.token

**Type:** string; use `"otp"` to enable the token prompt.

**Default:** no token prompt.

**Applies to:** `anyconnect` and `gp` password authentication.

Prompts for a YubiKey OTP through the proxy's menu or a notification reply. PACman appends the entered token to the configured password. The recognized value is `otp`; `totp` does not enable this prompt.

**Example:**

```yaml
options:
  token: otp
```

##### proxies.<name>.options.identity

**Type:** string containing a private key file path.

**Default:** no private key file.

**Applies to:** `ssh`.

Reads the private key from this file for SSH authentication. Use an absolute path; this setting does not expand `~`. Passphrase-protected private keys and the local SSH agent are not supported.

**Example:**

```yaml
options:
  identity: /Users/alice/.ssh/id_ed25519
```

### Rules

**Type:** list of rule objects.

**Default:** no explicit routing rules.

Each rule contains destination patterns in `hosts` and an ordered list of proxy names in `proxies`. Matching selects the most specific destination pattern, then tries that rule's proxies. An empty proxy list makes the matched destination connect directly.

**Example:**

```yaml
rules:
  - hosts: [".example.com"]
    proxies: [office, backup]
  - hosts: ["public.example.com"]
    proxies: []
```

#### rules[].hosts

**Type:** list of strings.

**Default:** no patterns; a rule without patterns matches no destinations.

Matches hostnames, exact IP addresses, or IP ranges in CIDR notation. Hostname matching is case-insensitive.

##### Hostnames and Wildcards

Use an exact hostname for one destination, or a wildcard to include subdomains. A leading dot is a convenient way to include both the domain itself and its subdomains. Quote patterns that start with `*` in YAML.

| Match type | Pattern | example.com | www.example.com |
|------------|---------|-------------|-----------------|
| Subdomains | `*.example.com` | No | Yes |
| Exact hostname | `example.com` | Yes | No |
| Domain and subdomains | `.example.com` | Yes | Yes |

Exact hostnames take precedence over wildcard patterns. More specific subdomain patterns take precedence over broader ones: `*.team.example.com` wins over `*.example.com`. Later rules replace earlier rules for identical hostname patterns.

##### IP Addresses and CIDR Ranges

Use an exact IP address, such as `192.168.1.20`, to match one destination. CIDR notation describes a whole network: `192.168.1.0/24` covers addresses from `192.168.1.0` through `192.168.1.255`. The number after `/` is the prefix length; a larger number describes a smaller, more specific range.

Narrower ranges take precedence over broader ones. For example, `192.168.1.0/26` takes precedence over `192.168.1.0/24` for addresses in the smaller range. Exact IP matches take precedence over CIDR ranges.

CIDR rules match IP addresses. If you want to match a site by its name, use a hostname or wildcard pattern; do not assume PACman will resolve every hostname to check it against a CIDR rule.

**Example:**

```yaml
hosts:
  - "*.internal.example.com"
  - ".example.com"
  - "192.168.1.0/24"
```

#### rules[].proxies

**Type:** list of strings naming entries in `proxies`.

**Default:** empty list (direct connection).

Tries the named proxies in list order until a connection succeeds. These are fallback choices; listing multiple proxies does not create a chain. An empty list bypasses proxying for the matched destinations, which is useful for exclusions.

**Example:**

```yaml
proxies:
  - office
  - backup
```

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
      token: otp
      timeout: "0"
  global_protect:
    username: admin
    password: secret
    protocol: gp
    host: gateway.example.com
    path: employee-group
    options:
      token: otp
      timeout: "7200"
  socks_proxy:
    username: proxyuser
    password: proxypass
    protocol: socks5
    host: proxy.example.com:1080
    options:
      timeout: "1800"
  ssh_tunnel:
    username: sshuser
    password: sshpass
    protocol: ssh
    host: proxy.example.com:22
    options:
      identity: /path/to/privatekey
      timeout: "0"

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

SSH checks if target host (`%h`) matches `rules[].hosts` via `pacman check`. If matched, traffic routes through PACman at `127.0.0.1:11078` (or custom address if `listen` changed) using a jump server.

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

### Runtime Diagnostics

PACman serves Go pprof under `/debug/pprof/` on the same address as `/proxy.pac`.

Open the endpoint index:

```bash
open http://127.0.0.1:11078/debug/pprof/
```

Start the pprof web UI for a CPU profile:

```bash
go tool pprof -http=:0 'http://127.0.0.1:11078/debug/pprof/profile?seconds=30'
```

Change `seconds` in the URL to control the CPU profile duration.
