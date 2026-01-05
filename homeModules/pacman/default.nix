{ config, lib, pkgs, ... }:

with lib;

let
  inherit(import ../.. { inherit pkgs; }) pacman;

  cfg = config.programs.pacman;

  rulefile = let
    toQs = attrs: concatStringsSep "&" (
      mapAttrsToList (k: v: "${escapeURL k}=${escapeURL v}") attrs
    );

    toProxyUrl = proxy: with proxy; concatStrings [
      (type + "://")
      (optionalString (username != null) (escapeURL username))
      (optionalString (password != null) (":" + escapeURL password))
      (optionalString (username != null || password != null) "@")
      address
      (optionalString (port != null) (":" + builtins.toString port))
      (optionalString (options != {}) ("/?" + toQs options))
    ];

    rulefile = (pkgs.formats.yaml {}).generate "ruleset" {
      listen = "${cfg.address}:${builtins.toString cfg.port}";
      proxies = mapAttrs (_: toProxyUrl) cfg.proxies;
      rules = cfg.rules;
    };
  in rulefile;

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
          type = types.listOf types.str;
        };
      };
    };

    proxy = types.submodule {
      options = {
        type = mkOption {
          type = types.enum ["http" "https" "socks5" "ssh" "anyconnect" "gp"];
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
          type = types.nullOr types.port;
          description = ''
            The port of the proxy to connect to.
          '';
          default = null;
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

        options = mkOption {
          type = types.attrsOf types.str;
          description = "Extra options for specific proxies";
          default = {};
        };
      };
    };
  };

  app = let
    appsDir =
      if config.targets.darwin.copyApps.enable
      then "${config.home.homeDirectory}/${config.targets.darwin.copyApps.directory}"
      else if config.targets.darwin.linkApps.enable
      then "${config.home.homeDirectory}/${config.targets.darwin.linkApps.directory}"
      else "${pacman}/Applications";
    in "${appsDir}/PACman.app/Contents/MacOS/PACman";
in {
  options.programs.pacman = {
    enable = mkEnableOption "PACman - Rule-based HTTP proxy server";

    ssh_config = mkOption {
      description = ''Enable ssh config management'';
      default = config.programs.ssh.enable;
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

    proxies = mkOption {
      description = ''
        Map of proxy keys to proxy definitions.
      '';
      type = types.attrsOf types.proxy;
      default = {};
    };

    rules = mkOption {
      description = ''
        Proxy rule definitions.
      '';
      type = types.listOf types.rule;
      default = [];
    };
  };

  config = mkIf cfg.enable (with cfg; {
    assertions = [
      {
        assertion = lib.all (rule:
          lib.all (name: builtins.hasAttr name cfg.proxies) rule.proxies
        ) cfg.rules;
        message = "All proxy names in rules must exist in top-level `proxies`.";
      }
    ];

    xdg.configFile."pacman/config" = {
      source = "${rulefile}";
      onChange = ''
        run /bin/launchctl kill SIGHUP gui/$(id -u)/io.github.gilliginsisland.pacman || true
      '';
    };

    launchd.agents.pacman = {
      enable = true;
      config = {
        Label = "io.github.gilliginsisland.pacman";
        KeepAlive = true;
        ProcessType = "Interactive";
        ProgramArguments = [
          "${app}" "proxy" "--launchd"
        ] ++ optionals (loglevel != null) [
          "--verbosity" loglevel
        ];
        Sockets = {
          Socket = {
            SockNodeName = address;
            SockServiceName = builtins.toString port;
          };
        };
        StandardOutPath = "${config.home.homeDirectory}/Library/Logs/${config.launchd.agents.pacman.config.Label}.log";
        StandardErrorPath = "${config.home.homeDirectory}/Library/Logs/${config.launchd.agents.pacman.config.Label}.log";
      };
    };

    programs.ssh.matchBlocks.pacman = mkIf cfg.ssh_config {
      match = ''exec "'${app}' check '%h'"'';
      proxyCommand = "${meta.getExe pkgs.netcat} -X 5 -x ${address}:${builtins.toString port} %h %p";
    };

    home.packages = [pacman];
  });
}
