{
  description = "Packages related to on demand AnyConnect VPNs";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs nixpkgs.lib.systems.flakeExposed;
      loadModulePaths = path: with builtins; mapAttrs (name: value: "${path}/${name}") (readDir path);
      pkgs = forAllSystems (system:
        if (nixpkgs.legacyPackages.${system}.config.allowUnfree)
        then nixpkgs.legacyPackages.${system}
        else import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        });
    in {
      packages = forAllSystems (system: import ./. {
        pkgs = pkgs.${system};
      });
      legacyPackages = forAllSystems (system: self.packages.${system} // {
        pkgs = self.legacyPackages.${system};
        pkgsStatic = {
          pkgs = self.legacyPackages.${system}.pkgs;
          pkgsStatic = self.legacyPackages.${system}.pkgsStatic;
        } // (import ./. {
          pkgs = pkgs.${system}.pkgsStatic;
        });
      });
      homeModules = loadModulePaths "${self}/homeModules";
      devShells = forAllSystems (system: {
        pacman = pkgs.${system}.mkShell {
          inputsFrom = [ self.packages.${system}.pacman ];
        };
      });
    };
}
