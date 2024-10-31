{
  description = "Packages related to on demand AnyConnect VPNs";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs nixpkgs.lib.systems.flakeExposed;
    in {
      packages = forAllSystems (system: import ./. {
        pkgs = nixpkgs.legacyPackages.${system};
      });
    };
}
