{ config, lib, pkgs, ... }:

with lib;

let
  inherit(import ../.. { inherit pkgs; }) pacman connect;

  cfg = config.programs.pacman;

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
  options.programs.pacman = {
    enable = mkEnableOption "PACman - Rule-based HTTP proxy server";

    ssh_config = mkOption {
      description = ''Enable ssh config management'';
      default = true;
      type = types.bool;
    };

    address = mkOption {
      description = ''
        The address to bind the PACman server.
        Use 0.0.0.0 to bind all interfaces.
      '';
      default = "127.0.0.1";
      type = types.str;
    };

    port = mkOption {
      description = ''
        The port to bind the PACman server.
      '';
      type = types.port;
    };

    loglevel = mkOption {
      description = ''
        Set the log level.
      '';
      default = null;
      type = types.nullOr (types.enum ["DEBUG" "INFO" "WARN" "ERROR"]);
    };

    rules = mkOption {
      description = ''
        Proxy rule definitions.
      '';
      type = types.listOf types.rule;
      default = [];
    };
  };

  config = mkIf cfg.enable {
    launchd.agents.pacman = {
      enable = true;
      config = {
        KeepAlive = true;
        ProcessType = "Background";
        ProgramArguments = with cfg; [
          (meta.getExe pacman)
          "-f" "${rulefile}"
          "-l" "${address}:${builtins.toString port}"
        ] ++ optionals (loglevel != null) [
          "-v" loglevel
        ];
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
        in {
          "proxypac:${toString i}" = {
            host = builtins.concatStringsSep " " hosts;
            proxyCommand = with cfg; "${meta.getExe connect} -R remote -H ${address}:${builtins.toString port} %h %p";
          };
        }
      )
      (filter (rule: rule.hosts != []) cfg.rules)
    ));
  };
}
