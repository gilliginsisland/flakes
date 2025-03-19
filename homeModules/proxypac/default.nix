{ config, lib, pkgs, ... }:

with lib;

let
  inherit(import ../.. { inherit pkgs; }) launch-socket-server single-serve pacman;

  cfg = config.programs.proxypac;

  pacfile = let
    rules' = reverseList (sortOn (r: stringLength r.host) (concatMap
      (rule: map
        (host: removeAttrs (rule // { inherit host; }) ["hosts"])
        rule.hosts
      )
      cfg.rules
    ));

    toProxyDirectives = proxies: concatMapStringsSep "; "
      (proxy: with proxy; "${toUpper type} ${address}:${builtins.toString port}")
      (concatMap
        (proxy: [proxy] ++ optionals
          (proxy.type == "socks5")
          [(proxy // { type = "socks"; })]
        )
        proxies
      );

    toShExpMatch = rule: with rule; ''
      if (shExpMatch(host, "${host}")) {
        return "${toProxyDirectives proxies}";
      }
    '';
  in pkgs.writeText "proxypac" ''
    function FindProxyForURL(url, host) {
      ${concatMapStrings toShExpMatch rules'}
      return 'DIRECT';
    }
  '';

  rulefile = let
    toProxyUrl = proxy: with proxy; concatStrings [
      (type + "://")
      (optionalString (username != null) (escapeURL username))
      (optionalString (password != null) (":" + escapeURL password))
      (optionalString (username != null) "@")
      address
      (optionalString (port != null) (":" + builtins.toString port))
      (optionalString (identity != null) "/?identity=${escapeURL identity}")
    ];

    pacManRules = builtins.toJSON (map
      (rule: rule // {
        proxies = map toProxyUrl rule.proxies;
      })
      cfg.rules
    );
  in pkgs.writeText "rulefile" pacManRules;

  types = lib.types // rec {
    rule = types.submodule {
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
          type = types.listOf proxy;
        };
      };
    };

    proxy = types.submodule {
      options = {
        type = mkOption {
          type = types.enum ["http" "https" "socks5" "ssh"];
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

        username = mkOption {
          type = types.nullOr types.str;
          description = "Username for authentication";
          default = null;
        };

        password = mkOption {
          type = types.nullOr types.str;
          description = "Password for authentication";
          default = null;
        };

        identity = mkOption {
          type = types.nullOr types.path;
          description = "Path to the private key file for authentication";
          default = null;
        };
      };
    };
  };
in {
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

    pacman = {
      enable = mkEnableOption "pacman - Rule-based HTTP proxy server";
      address = mkOption {
        description = ''
          The address to bind the pacman server.
        '';
        default = "127.0.0.1";
        type = types.str;
      };
      port = mkOption {
        description = ''
          The port to bind the pacman server.
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
      type = types.listOf types.rule;
      default = [];
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
        StandardErrorPath = "${config.home.homeDirectory}/Library/Logs/${config.launchd.agents.proxypac.config.Label}.log";
      };
    };

    launchd.agents.pacman = {
      enable =  cfg.pacman.enable;
      config = {
        KeepAlive = true;
        ProcessType = "Background";
        ProgramArguments = [
          (meta.getExe pacman)
          "-f" "${rulefile}"
          "-l" "${cfg.pacman.address}:${builtins.toString cfg.pacman.port}"
        ] ++ optionals cfg.pacman.debug [ "-v" ];
        # Sockets = {
        #   Socket = {
        #     SockNodeName = cfg.pacman.address;
        #     SockServiceName = builtins.toString cfg.pacman.port;
        #   };
        # };
        StandardErrorPath = "${config.home.homeDirectory}/Library/Logs/${config.launchd.agents.pacman.config.Label}.log";
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
