{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.proxypac;

  cgiPkg = pkgs.writeShellApplication {
    name = "proxypac-cgi";
    text = ''
      CONTENT="''$(<"''${1}")"
      LENGTH=''${#CONTENT}

      echo "HTTP/1.1 200 OK"
      echo "Content-Type: application/x-ns-proxy-autoconfig"
      echo "Content-Length: ''${LENGTH}"
      echo "Connection: close"
      echo ""
      echo -en "''${CONTENT}"

      exit 0
    '';
  };

  toPAC = rules: ''
    const rules = ${builtins.toJSON rules};

    const entries = rules.flatMap(
      ({ hosts, proxy }) => hosts.map(host => [host, proxy])
    ).sort(
      ([a], [b]) => b.length - a.length
    );

    function FindProxyForURL(_, host) {
      for (const [shExp, proxy] of entries) {
        if (shExpMatch(host, shExp)) {
          const { type, address, port } = proxy;

          if (type === "socks5") {
            return ["SOCKS5", "SOCKS"].map(scheme => `''${scheme} ''${address}:''${port}`).join(";");
          }
        }
      }
      return 'DIRECT';
    }
  '';
in
{
  options.programs.proxypac = {
    enable = mkEnableOption "Proxy Auto Configuration";

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

          proxy = mkOption {
            description = ''
              Proxy configuration.
            '';
            type = types.submodule {
              options = {
                type = mkOption {
                  type = types.enum [ "socks5" ];
                  default = "socks5";
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
          (meta.getExe cgiPkg)
          "${config.xdg.configHome}/proxypac/proxy.pac"
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

    xdg.configFile."proxypac/proxy.pac".text = toPAC cfg.rules;

    programs.ssh.matchBlocks = listToAttrs (imap1 (n: rule:
      let
        inherit (rule) hosts;
        inherit (rule.proxy) address port;
      in
        nameValuePair "proxypac:${builtins.toString n}" {
          host = builtins.concatStringsSep " " hosts;
          proxyCommand = "${meta.getExe pkgs.netcat} -X 5 -x ${address}:${builtins.toString port} %h %p";
        }
    ) cfg.rules);
  };
}
