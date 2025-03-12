{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.proxypac;

  rules = reverseList (sortOn (r: stringLength r.hosts) (concatMap
    (rule: map
      (host: (rule // { hosts = host; }))
      rule.hosts
    )
    cfg.rules
  ));

  toProxyPacDirective = proxies: concatMapStringsSep "; "
    (proxy: with proxy; "${toUpper type} ${address}:${builtins.toString port}")
    (concatMap
      (proxy: [proxy] ++ optionals
        (proxy.type == "socks5")
        [(proxy // { type = "socks"; })]
      )
      proxies
    );

  toIf = rule: with rule; ''
    if (shExpMatch(host, "${hosts}")) {
      return "${toProxyPacDirective proxies}";
    }
  '';

  pacfile = pkgs.writeText "proxypac" ''
    function FindProxyForURL(url, host) {
    ${concatMapStrings toIf rules}
    return 'DIRECT';
    }
  '';

  single-serve = pkgs.callPackage ../../packages/single-serve/package.nix {};
  pacproxy = pkgs.callPackage ../../packages/pacproxy.nix {};
in
{
  options.programs.proxypac = {
    enable = mkEnableOption "Proxy Auto Configuration";

    ssh_config = mkOption {
      description = ''Enable ssh config management'';
      default = true;
      type = types.bool;
    };

    address = mkOption {
      description = ''
        The address to bind the PAC server to.
        Use 0.0.0.0 to bind all interfaces.
      '';
      default = "127.0.0.1";
      type = types.str;
    };

    port = mkOption {
      description = ''
        The port to bind the PAC server to.
      '';
      type = types.port;
    };

    http = {
      enable = mkEnableOption "HTTP proxypac Wrapper";
      address = mkOption {
        description = ''
          The address to bind the HTTP proxypac server.
        '';
        default = "127.0.0.1";
        type = types.str;
      };
      port = mkOption {
        description = ''
          The port to bind the HTTP proxypac server.
        '';
        type = types.port;
      };
      debug = mkOption {
        description = ''
          Enable debug output.
        '';
        default = false;
        type = types.bool;
      };
    };

    rules = mkOption {
      description = ''
        Proxy rule definitions. Will be used to build the proxy.pac file.
      '';
      type = types.listOf (types.submodule {
        options = {
          hosts = mkOption {
            description = ''
              List of domain patterns to route through the proxy.
            '';
            type = types.listOf types.str;
          };

          proxies = mkOption {
            description = ''
              Proxy configuration.
            '';
            type = types.listOf (types.submodule {
              options = {
                type = mkOption {
                  type = types.enum ["http" "https" "socks5"];
                  description = "The type of proxy.";
                };

                address = mkOption {
                  type = types.str;
                  default = "127.0.0.1";
                  description = ''
                    The address of the proxy to connect to.
                  '';
                };

                port = mkOption {
                  type = types.port;
                  description = ''
                    The port of the proxy to connect to.
                  '';
                };
              };
            });
          };
        };
      });
    };

    text = mkOption {
      type = types.lines;
      internal = true;
    };
  };

  config = mkIf cfg.enable {
    launchd.agents.proxypac = {
      enable = true;
      config = {
        ProcessType = "Background";
        ProgramArguments = [
          (meta.getExe single-serve) "${pacfile}" "application/x-ns-proxy-autoconfig"
        ];
        inetdCompatibility.Wait = false;
        Sockets = {
          Socket = {
            SockNodeName = cfg.address;
            SockServiceName = builtins.toString cfg.port;
          };
        };
        StandardErrorPath = "${config.xdg.stateHome}/proxypac/proxypac.log";
      };
    };

    launchd.agents.pacproxy = mkIf cfg.http.enable {
      enable =  true;
      config = {
        KeepAlive = true;
        ProcessType = "Background";
        ProgramArguments = [
          (meta.getExe pkgs.pacproxy)
          "-c" "${pacfile}"
          "-l" "${cfg.http.address}:${builtins.toString cfg.http.port}"
        ] ++ optionals cfg.http.debug [ "-v" ];
        StandardErrorPath = "${config.xdg.stateHome}/proxypac/http.log";
      };
    };

    programs.ssh.matchBlocks = mkIf cfg.ssh_config (mergeAttrsList (imap1
      (i: rule:
        let
          inherit (rule) hosts;
          proxy = builtins.elemAt rule.proxies 0;
          inherit (proxy) address port;
        in {
          "proxypac:${toString i}" = {
            host = builtins.concatStringsSep " " hosts;
            proxyCommand = "${meta.getExe pkgs.netcat} -X 5 -x ${address}:${builtins.toString port} %h %p";
          };
        }
      )
      (filter (rule: rule.hosts != []) cfg.rules)
    ));
  };
}
