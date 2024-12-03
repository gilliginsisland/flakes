{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.ocmanager;

  profileModule = types.submoduleWith {
    modules = [ ./profile.nix ];
    specialArgs = { inherit pkgs; };
    shorthandOnlyDefinesConfig = true;
  };

  toLaunchd =  name: profile: {
    ProcessType = "Background";
    ProgramArguments = [
      "${pkgs.launch_socket_server}/bin/launch_socket_server"
      "${pkgs.ocmanager}/bin/ocmanager"
      "-p"
      "${name}"
    ];
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
      type = types.attrsOf profileModule;
      default = { };
      description = ''
        Profiles for ocmanager.
      '';
    };
  };

  config = mkIf cfg.enable {
    home.packages = with pkgs; [ ocmanager ];

    launchd.agents = concatMapAttrs (name: profile: optionalAttrs profile.proxy.ondemand {
      "ocmanager.${name}" = {
        enable = true;
        config = toLaunchd name profile;
      };
    }) cfg.profiles;

    xdg.configFile = mapAttrs' (name: profile:
      nameValuePair "ocmanager/profiles/${name}.conf" {
        inherit (profile) text;
      }
    ) cfg.profiles;

    programs.proxypac.rules = mapAttrsToList (
      name: profile: {
        inherit (profile) hosts;
        proxy = {
          type = "socks5";
          address = "127.0.0.1";
          inherit (profile.proxy) port;
        };
      }
    ) cfg.profiles;
  };
}
