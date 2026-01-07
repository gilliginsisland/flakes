{
  description = "Packages related to on demand AnyConnect VPNs";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs nixpkgs.lib.systems.flakeExposed;
      loadModulePaths = path: with builtins; mapAttrs (name: value: "${path}/${name}") (readDir path);
    in {
      packages = forAllSystems (system: import ./. {
        pkgs = nixpkgs.legacyPackages.${system};
      });
      legacyPackages = forAllSystems (system: self.packages.${system} // {
        pkgs = self.legacyPackages.${system};
        pkgsStatic = self.legacyPackages.${system} // (import ./. {
          pkgs = nixpkgs.legacyPackages.${system}.pkgsStatic;
        });
      });
      homeModules = loadModulePaths "${self}/homeModules";
    };
}
