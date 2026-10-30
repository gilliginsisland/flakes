# v3.3.0

* Redesigned the proxy menu with live connection status and Connect, Cancel, Disconnect, and Reset controls
* Enter YubiKey one-time passwords directly in each proxy's menu or through notification replies
* Added menu bar animations for connection progress, authentication prompts, and connection failures
* Reset failed proxy connections directly from their notifications
* Added searchable offline Help with expanded setup instructions and configuration examples
* Preserve unchanged proxy connections when reloading settings, and keep the current configuration if a rule refers to a missing proxy
* Proxy options such as timeout now accept numbers without quotes in YAML and JSON configuration
* Fixed inconsistent hostname matching between proxy routing and SSH checks, including mixed-case names and proxy aliases
* Improved handling of HTTP, SOCKS5, and SSH connections sharing the same proxy port
* Improved VPN packet transfer performance by replacing socket pairs with shared-memory ring buffers
* Improved VPN setup, cancellation, disconnection, and error reporting, including stopping repeated attempts after GlobalProtect rejects authentication
___
# v3.2.0

* Updated hostscan definitions to work with newer cisco versions
___
# v3.1.9

* Signed release builds with a stable certificate for more reliable automatic updates
___
# v3.1.8

* Optimized posture checking and VPN packet handling
* Improved VPN error and status notifications
* Added runtime diagnostics for troubleshooting connection issues
* Fixed proxy aliases after config reloads and with mixed-case hostnames
* Updated dependencies
___
# v3.1.7

* bugfix: send Cisco AnyConnect client version in auth requests
___
# v3.1.6

* bugfix: write CONNECT tunnel responses without transfer encoding
* chore: update deps and toolchain to go1.26
___
# v3.1.5

* feat: create default config on launch
___
# v3.1.4

* feat: major networking performance improvement
___
# v3.1.3

* bugfix: nil pointer on dynamic item
___
# v3.1.2

* feat: improve packet handling performance
___
# v3.1.1

* feat: improve packet handling performance
* feat: integrate Sparkle for automatic updates
___
# v3.1.0

* feat: redesigned menu with status badges
