{ pkgs }:

let
  callPackage = pkgs.lib.callPackageWith (pkgs // impure // packages);
  impure = pkgs.lib.callPackageWith pkgs ./impure-cmds.nix { };
  packages = {
    launch_socket_server = callPackage ./launch_socket_server { };
    ocmanager = callPackage ./ocmanager { };
    yksofttoken = callPackage ./yksofttoken { };
  };
in
  packages
