{ pkgs }:

let
  internal = pkgs.lib.callPackagesWith pkgs ./internal { };
  callPackage = pkgs.lib.callPackageWith (pkgs // internal // packages);
  packages = pkgs.lib.packagesFromDirectoryRecursive {
    inherit callPackage;
    directory = ./packages;
  };
in
  packages
