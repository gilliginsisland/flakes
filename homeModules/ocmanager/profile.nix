{ config, lib, pkgs, ... }:

with lib;

let
  toProxyScript = { ondemand, address, port }:
    let
      bindArg = if ondemand then ''''${LAUNCH_CMD_ADDRESS}'' else "${address}:${port}";
    in ''
      "${pkgs.ocproxy}/bin/ocproxy" -D "${bindArg}"
    '';

  mkKeyValue = key: value:
    if builtins.isBool value then (
      if value then key else ""
    ) else "${key}=${builtins.toString value}";

  toText = generators.toKeyValue { inherit mkKeyValue; };
in {
  options = {
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

    config = mkOption {
      type = types.attrsOf (types.either types.str types.bool);
      default = {};
    };

    text = mkOption {
      type = types.lines;
      default = "";
      internal = true;
    };
  };

  config = mkMerge [
    (mkIf (config.proxy != null) {
      config = {
        script-tun = true;
        script = toProxyScript { inherit (config.proxy) ondemand address port; };
      };
    })

    {
      text = toText config.config;
    }
  ];
}
