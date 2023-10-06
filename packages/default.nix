{ pkgs }:

let
  callPackage = pkgs.lib.callPackageWith (pkgs // impure // packages);
  impure = with callPackage ./impure-cmds.nix { }; {
    inherit security;
    inherit osascript;
    inherit curl;
  };
  packages = {
    launch_socket_server = callPackage ./launch_socket_server { };
    ocmanager = callPackage ./ocmanager { };
  };
in
  packages
