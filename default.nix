{ pkgs }:

let
  callPackage = pkgs.lib.callPackageWith (pkgs // packages // { inherit callPackage; });
  packages = pkgs.lib.packagesFromDirectoryRecursive {
    inherit callPackage;
    directory = ./packages;
  };
in
  packages
