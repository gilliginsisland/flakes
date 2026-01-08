# PACman - Proxy Auto Configuration Manager

PACman is a proxy management tool designed to simplify **proxy** and **VPN management**. It runs a **SOCKS5** and **HTTP rule-based proxy** on **localhost port 11078**. It also serves a **Proxy PAC file** for optimizing browser traffic. It supports **Cisco AnyConnect** and **GlobalProtect proxies** using the **openconnect library**, as well as **HTTP** and **SOCKS5 proxies** via **golang.org/x/net/proxy**.

## How It Works

PACman acts as a local proxy server that intercepts incoming traffic and routes it to upstream proxies based on rules defined in a configuration file. When a request matches a rule, PACman attempts to forward the traffic through the list of proxies specified in the rule, trying each in order until a successful connection is established.

Additionally, PACman manages VPN connections internally, ensuring that protocols like Cisco AnyConnect and GlobalProtect are properly initialized and maintained during operation.

### DNS Resolution

PACman respects `/etc/hosts` for DNS resolution, checking it first before delegating to the underlying proxy for further resolution. If a hostname is mapped in `/etc/hosts`, PACman will instruct the proxy to connect directly to the specified IP address to honor the local configuration. If no rule matches the unmapped hostname, PACman will check the resolved IP address against the rules for a potential match (e.g., against CIDR patterns).

### Rule Matching Logic

PACman uses a **most-specific-match-first** approach for rule matching. The specificity of a rule determines its priority, and the order of rules in the configuration file is only considered when two rules have the same specificity. The specificity hierarchy is as follows:
- For domain matches:
  - An **exact match** (e.g., `example.com`) or a **specific subdomain** (e.g., `sub.example.com`) takes precedence over wildcards.
  - For zone/wildcard matches (e.g., `*.example.com`), a **longer match** (e.g., `*.sub.example.com`) is considered more specific than a shorter one (e.g., `*.example.com`).
- For CIDR matches:
  - A **smaller CIDR range** (e.g., `192.168.1.0/26`) is considered more specific than a larger one (e.g., `192.168.1.0/24`).

### Proxy Chaining and Circular Reference Caution

PACman supports **proxy chaining**, allowing a proxy to be routed through another proxy if it matches a rule. For example, a rule might specify an SSH proxy that itself requires a Cisco AnyConnect VPN; PACman will handle this nested routing. However, **care must be taken to avoid circular references** (e.g., Proxy A pointing to Proxy B, which points back to Proxy A), as the application does not protect against such loops and they may cause failures or crashes.

## Rule File Configuration

PACman uses a YAML (or JSON, depending on your setup) configuration file to define proxies and routing rules. Below is the structure of the rule file:

### Config File Format

The configuration `~/.config/pacman/config` consists of a root object with three primary keys: `listen`, `proxies`, and `rules`.

- **`listen`**: Specifies the host and port on which PACman will listen for incoming connections. This is in the format `host:port` (e.g., `127.0.0.1:11078`). If not specified, it defaults to `127.0.0.1:11078`.

- **`proxies`**: A map where:
  - The key is a unique label for the proxy (used in the UI status dropdown and for referencing in rules).
  - The value is a URI string specifying the proxy endpoint using standard URI format. The URI must include username and password (if required) directly in the scheme, following the format `scheme://username:password@host:port`.
  
- **`rules`**: An array of rule objects, where each rule defines:
  - `hosts`: An array of strings representing host match patterns. These patterns can be:
    - **Zone/Wildcard Match**: Using a leading asterisk (e.g., `*.example.com`) matches subdomains like `www.example.com` but **not** the base domain `example.com`.
    - **Exact Match**: A plain domain (e.g., `example.com`) matches only the exact domain `example.com` and **not** subdomains like `www.example.com`.
    - **Leading Dot Notation**: A leading dot (e.g., `.example.com`) matches **both** the base domain `example.com` and all subdomains like `www.example.com`.
    - **CIDR Notation**: An IP range (e.g., `192.168.1.0/24`) matches hosts within the specified network range.
  - `proxies`: An array of proxy labels (referencing keys from the `proxies` map). For a matching host, PACman will attempt to connect through each proxy in the list, in order, until successful.
    
    **Note**: An empty `proxies` array means to skip proxying for the matched hosts, allowing you to exclude subsets (e.g., define a broad CIDR to use a proxy, then exclude a smaller CIDR subset by setting an empty `proxies` list for it).

### Example Configuration

```yaml
listen: 127.0.0.1:11078
proxies:
  cisco_vpn: anyconnect://user:pass@vpn.example.com/usergroup?token=totp&timeout=0
  global_protect: gp://admin:secret@gateway.example.com/employee-group?token=totp&timeout=7200
  socks_proxy: socks5://proxyuser:proxypass@proxy.example.com:1080?timeout=1800
  ssh_tunnel: ssh://sshuser:sshpass@proxy.example.com:22?identity=/path/to/privatekey&timeout=0

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

In this example:
- Traffic to `*.internal.example.com`, `.example.com`, or the CIDR range `192.168.1.0/24` will first attempt to route through `cisco_vpn`, falling back to `socks_proxy` if it fails.
- Traffic to `specific.example.com` will try `global_protect` first, then `ssh_tunnel`.
- Each proxy URI includes the username and password directly in the format `scheme://username:password@host:port` (or `scheme://username:password@host` if no port is specified).

## Supported Proxy Protocols

PACman supports the following proxy types with specific configuration options passed via query parameters or URI paths where applicable:

- **Global Options for All Proxy Types**:
  - `timeout=<seconds>`: Sets the idle timeout in seconds. Default is 3600 (1 hour). Set to `0` to disable idle disconnecting.

- **Cisco AnyConnect**:
  - **Scheme**: `anyconnect://`
  - **Usergroup**: Specify the usergroup in the URI path (e.g., `anyconnect://user:pass@vpn.example.com/usergroup`).
  - **Additional Options**:
    - `token=totp`: Prompts an alert box to input a YubiKey TOTP token, which is appended to the password during connection.

- **Palo Alto Networks GlobalProtect**:
  - **Scheme**: `gp://`
  - **Usergroup**: Specify the usergroup in the URI path (e.g., `gp://admin:secret@gateway.example.com/employee-group`).
  - **Additional Options**:
    - `token=totp`: Prompts an alert box to input a YubiKey TOTP token, which is appended to the password during connection.

- **SSH Proxy**:
  - **Scheme**: `ssh://`
  - **Additional Options**:
    - `identity=<path>`: Specifies the path to a private key file for authentication (e.g., `ssh://user:pass@proxy.example.com:22?identity=/path/to/privatekey`). Note: Files with passphrases are not supported, and the local SSH agent is not used for authentication.

- **SOCKS5 Proxy**:
  - **Scheme**: `socks5://` or `socks5h://`
  - **Description**: Supports SOCKS5 proxies with or without hostname resolution (e.g., `socks5://user:pass@proxy.example.com:1080`).

- **HTTP/HTTPS Proxy**:
  - **Scheme**: `http://` or `https://`
  - **Description**: Supports standard HTTP or HTTPS proxies (e.g., `http://user:pass@proxy.example.com:8080`).

**Note**: For proxies requiring authentication, the username and password must be embedded in the URI as `scheme://username:password@host:port`. Ensure these credentials are securely managed and not exposed in version control or unsecured environments. By default, connections are lazy (only established when needed) and disconnect after 1 hour of inactivity unless modified via the `timeout` parameter.

## How to Use

This section provides guidance on enabling and configuring PACman for various applications and use cases.

### Enabling the Proxy

PACman listens on the following addresses by default (configurable via the `listen` field in the configuration file):
- **HTTP Proxy**: `http://127.0.0.1:11078`
- **SOCKS5 Proxy**: `socks5://127.0.0.1:11078`

You can configure applications to use either the HTTP or SOCKS5 proxy directly, or utilize the PAC file for automatic configuration (see Proxy PAC File Support below).

### Browser Configuration on macOS

To configure your browser on macOS to use PACman, you have two options:

1. **Using the PAC File (Recommended for Performance)**:
   - Open your browser's proxy settings.
   - Set the "Automatic Proxy Configuration" URL to `http://127.0.0.1:11078/proxy.pac` (or the custom address if you've changed the `listen` setting in the configuration).
   - This allows the browser to route only matching traffic through PACman, improving performance for non-matching traffic.

2. **Using the SOCKS5 Proxy Directly**:
   - Alternatively, configure the browser to use a manual SOCKS5 proxy.
   - Set the SOCKS5 proxy host to `127.0.0.1` and port to `11078` (or the custom address if you've changed the `listen` setting in the configuration).
   - This routes all browser traffic through PACman, which may impact performance for non-matching traffic.

For system-wide proxy settings on macOS (affecting most browsers and applications):
- Go to **System Settings > Network > [Your Active Network] > Details > Proxies**.
- Enable "Automatic Proxy Configuration" and enter `http://127.0.0.1:11078/proxy.pac` (or the custom address if configured), or manually set the SOCKS5 proxy to `127.0.0.1:11078` (or the custom port).

### Proxy PAC File Support

PACman serves a Proxy Auto-Config (PAC) file on the same HTTP port at `http://127.0.0.1:11078/proxy.pac` (or the custom address if you've changed the `listen` setting in the configuration). This PAC file routes traffic with matching rules directly to the PACman proxy. For browsers and other applications that support PAC files, this can improve performance by avoiding unnecessary routing through PACman for non-matching traffic. Configure your browser or application to use this PAC file URL for automatic proxy configuration.

### SSH Integration

PACman can be integrated with SSH to automatically route traffic through the proxy for matching hosts. Add the following configuration to your `~/.ssh/config` file to enable this functionality:

```
Match exec "'/Applications/PACman.app/Contents/MacOS/pacman' check '%h'"
    ProxyCommand nc -X 5 -x 127.0.0.1:11078 %h %p
```

This configuration checks if the target host (`%h`) matches a rule in PACman using the `pacman check` command. If it matches, SSH will use the PACman proxy (on `127.0.0.1:11078` by default, or the custom address if configured) via `nc` with SOCKS5 (`-X 5`) to forward traffic to the target host and port (`%h %p`).

### Terminal and Other HTTP-Based Applications

For terminal applications or other tools that support HTTP proxies (e.g., `curl`, `wget`, or Rancher Desktop):

- Set the following environment variables in your terminal (e.g., in `~/.zshrc`, `~/.bashrc`, or equivalent):
  ```bash
  export HTTP_PROXY=http://127.0.0.1:11078  # or custom address if configured
  export HTTPS_PROXY=http://127.0.0.1:11078 # or custom address if configured
  export NO_PROXY=localhost,127.0.0.1
  ```
- For applications like Rancher Desktop that require explicit proxy settings:
  - Open the application’s settings or configuration file.
  - Set the HTTP and HTTPS proxy to `http://127.0.0.1:11078` (or the custom address if configured).
  - Add `localhost` and `127.0.0.1` to the "bypass proxy" or "no proxy" list to avoid local traffic routing issues.

For tools supporting SOCKS5 proxies, configure them to use `socks5://127.0.0.1:11078` (or the custom address if configured) if HTTP proxy settings are not supported.
