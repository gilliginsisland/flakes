{ config, lib, pkgs, ... }:

with lib;

let
  toProxyScript = { ondemand, address, port }:
    let
      bindArg = if ondemand then ''''${LAUNCH_CMD_ADDRESS}'' else "${address}:${port}";
    in ''
      "${getBin pkgs.ocproxy}/bin/ocproxy" -D "${bindArg}"
    '';

  mkKeyValue = key: value:
    if builtins.isBool value then (
      if value then key else ""
    ) else (
      if value != null then "${key}=${builtins.toString value}" else ""
    );

  toText = generators.toKeyValue { inherit mkKeyValue; };
in {
  options = {
    user = mkOption {
      description = ''
        Username.
      '';
      type = types.str;
    };

    authgroup = mkOption {
      default = null;
      description = ''
        Authgroup.
      '';
      type = types.nullOr types.str;
    };

    server = mkOption {
      description = ''
        The address / hostname of the server.
      '';
      type = types.str;
    };

    useragent = mkOption {
      default = null;
      description = ''
        The useragent to use.
      '';
      type = types.nullOr types.str;
    };

    protocol = mkOption {
      description = ''
        The address / hostname of the server.
      '';
      type = types.enum ["anyconnect" "gp"];
    };

    token = mkOption {
      default = false;
      description = ''
        If touch token support is required.
      '';
      type = types.bool;
    };

    hosts = mkOption {
      default = [];
      description = ''
        List of domain patterns to route through the proxy.
      '';
      type = types.listOf types.str;
    };

    proxy = mkOption {
      default = null;
      description = ''
        Convenience for configuring the VPN as a SOCKS proxy.
      '';
      type = types.nullOr (types.submodule {
        options = {
          ondemand = mkOption {
            type = types.bool;
            default = true;
            description = "Whether to run the proxy ondemand.";
          };

          address = mkOption {
            type = types.str;
            default = "127.0.0.1";
            description = ''
              The address to bind the SOCKS proxy to.
              Use 0.0.0.0 to bind all interfaces.
            '';
          };

          port = mkOption {
            type = types.port;
            description = ''
              The port to bind the SOCKS proxy to.
            '';
          };
        };
      });
    };

    extraConfig = mkOption {
      type = types.attrsOf (types.nullOr (types.either types.str types.bool));
      default = {};
    };

    text = mkOption {
      type = types.lines;
      internal = true;
    };
  };

  config = mkMerge [
    (mkIf (config.protocol == "anyconnect") {
      useragent = mkDefault "AnyConnect Darwin_i386 4.10.01075";
    })

    (mkIf (config.protocol == "gp") {
      useragent = mkDefault "Global Protect";
    })

    {
      extraConfig = mapAttrs (_: mkDefault) {
        inherit (config) user server useragent protocol authgroup;
      };
    }

    (mkIf (config.proxy != null) {
      extraConfig = {
        script-tun = true;
        script = toProxyScript {
          inherit (config.proxy) ondemand address port;
        };
      };
    })

    {
      text = toText config.extraConfig;
    }
  ];
}
