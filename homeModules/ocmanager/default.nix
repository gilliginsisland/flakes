{ config, lib, pkgs, ... }:

with lib;

let
  cfg = config.programs.ocmanager;

  inherit(import ../.. { inherit pkgs; }) launch-socket-server ocmanager;

  filterProfiles = f: filterAttrs (name: profile: f profile) cfg.profiles;

  toLaunchd = name: profile: {
    ProcessType = "Background";
    ProgramArguments = [
      (meta.getExe launch-socket-server)
      (meta.getExe ocmanager)
      "-c"
      ''${pkgs.writeTextDir name profile.text}/${name}''
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

  toPACRule = name: profile: {
    inherit (profile) hosts;
    proxies = [{
      type = "socks5";
      address = "127.0.0.1";
      inherit (profile.proxy) port;
    }];
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

    launchd.agents = mapAttrs'
      (name: profile: nameValuePair "ocmanager.${name}" {
        enable = true;
        config = toLaunchd name profile;
      })
      (filterProfiles (profile: profile.proxy.ondemand));

    programs.proxypac.rules = mapAttrsToList toPACRule
      (filterProfiles (profile: profile.hosts != []));
  };
}
