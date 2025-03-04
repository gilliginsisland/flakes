{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.ocmanager;

  inherit(import ../.. { inherit pkgs; }) launch-socket-server ocmanager;

  toLaunchd =  name: profile: {
    ProcessType = "Background";
    ProgramArguments = [
      (meta.getExe launch-socket-server)
      (meta.getExe ocmanager)
      "-p" "${name}"
    ] ++ optionals profile.token ["-t"];
    Sockets = {
      Socket = {
        SockNodeName = profile.proxy.address;
        SockServiceName = builtins.toString profile.proxy.port;
      };
    };
    StandardOutPath = "${config.xdg.stateHome}/ocmanager/${name}.log";
    StandardErrorPath = "${config.xdg.stateHome}/ocmanager/${name}.log";
  };
in {
  options.programs.ocmanager = {
    enable = mkEnableOption "OCManager configuration";
    package = mkPackageOption pkgs "ocmanager" { };
    profiles = mkOption {
      description = ''
        Profiles for ocmanager.
      '';
      default = { };
      type = types.attrsOf (types.submoduleWith {
        modules = [ ./profile.nix ];
        specialArgs = { inherit pkgs; };
        shorthandOnlyDefinesConfig = true;
      });
    };
  };

  config = mkIf cfg.enable {
    home.packages = [ ocmanager ];

    launchd.agents = concatMapAttrs
      (name: profile: optionalAttrs profile.proxy.ondemand {
        "ocmanager.${name}" = {
          enable = true;
          config = toLaunchd name profile;
        };
      })
      cfg.profiles;

    xdg.configFile = mapAttrs'
      (name: profile: nameValuePair "ocmanager/profiles/${name}.conf" {
        inherit (profile) text;
      })
      cfg.profiles;

    programs.proxypac.rules = concatMapAttrs
      (name: profile:
        let
          mkRule = type: hosts: {
            inherit hosts;
            proxy = {
              inherit type;
              inherit (profile.proxy) port;
              address = "127.0.0.1";
            };
          };
          inherit (profile) hosts resolve_hosts;
        in {
          ${"ocmanager:${name}:hosts"} = (mkRule "socks5h" hosts);
          ${"ocmanager:${name}:resolve_hosts"} = (mkRule "socks5" resolve_hosts);
        }
      )
      cfg.profiles;
  };
}
