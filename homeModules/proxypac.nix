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

    const entries = [].concat(...rules.map(
      ({ hosts, proxies }) => hosts.map(
        host => [host, proxies]
      )
    )).sort(
      ([a], [b]) => b.length - a.length
    )

    function FindProxyForURL(_, host) {
      for (const [shExp, proxies] of entries) {
        if (shExpMatch(host, shExp)) {
          return proxies.join(';');
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

          proxies = mkOption {
            description = ''
              Proxy destination lines.
            '';
            type = types.listOf types.str;
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
          "${cgiPkg}/bin/proxypac-cgi"
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
  };
}
