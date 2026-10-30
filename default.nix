{ pkgs, buildDate ? null }:

let
  callPackage = pkgs.lib.callPackageWith (pkgs // packages // { inherit callPackage buildDate; });
  packages = pkgs.lib.packagesFromDirectoryRecursive {
    inherit callPackage;
    directory = ./packages;
  };
in
  packages
