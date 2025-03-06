{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.proxypac;
  rules = lib.attrValues cfg.rules;

  single-serve = pkgs.callPackage ../../packages/single-serve/package.nix {};
  pacproxy = pkgs.callPackage ../../packages/pacproxy.nix {};

  pacfile = pkgs.writeText "proxypac" ''
    var rules = ${builtins.toJSON rules};

    var entries = [];
    rules.forEach(function(rule){
      rule.hosts.forEach(function(host){
        entries.push([host, rule.proxy]);
      });
    });
    entries.sort(function(a, b) {
      return b[0].length - a[0].length;
    });

    function toProxyDirective(proxy) {
      switch (proxy.type) {
        case "http":
          return "PROXY " + proxy.address + ":" + proxy.port;
        case "https":
          return "HTTPS " + proxy.address + ":" + proxy.port;
        case "socks5":
          return "SOCKS5 " + proxy.address + ":" + proxy.port + ";SOCKS " + proxy.address + ":" + proxy.port;
      }
      return proxy.type + " " + proxy.address + ":" + proxy.port;
    }

    function FindProxyForURL(_, host) {
      for (var i = 0; i < entries.length; i++) {
        var entry = entries[i];

        var shExp = entry[0];
        var proxy = entry[1];

        if (shExpMatch(host, shExp)) {
          return toProxyDirective(proxy);
        }
      }
      return 'DIRECT';
    }
  '';
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
      type = types.attrsOf (types.submodule {
        options = {
          hosts = mkOption {
            description = ''
              List of domain patterns to route through the proxy.
            '';
            type = types.listOf types.str;
          };

          proxy = mkOption {
            description = ''
              Proxy configuration.
            '';
            type = types.submodule {
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
            };
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
        ] ++ lib.optionals cfg.http.debug [ "-v" ];
        StandardErrorPath = "${config.xdg.stateHome}/proxypac/http.log";
      };
    };

    programs.ssh.matchBlocks = mkIf cfg.ssh_config (mapAttrs'
      (name: rule: nameValuePair "proxypac:${name}" (
        let
          inherit (rule) hosts;
          inherit (rule.proxy) address port;
        in {
          host = builtins.concatStringsSep " " hosts;
          proxyCommand = "${meta.getExe pkgs.netcat} -X 5 -x ${address}:${builtins.toString port} %h %p";
        }
      ))
      (filterAttrs (name: rule: rule.hosts != []) cfg.rules)
    );
  };
}
