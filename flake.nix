{
  description = "Packages related to on demand AnyConnect VPNs";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs nixpkgs.lib.systems.flakeExposed;
      loadModulePaths = path: with builtins; mapAttrs (name: value: "${path}/${name}") (readDir path);
      pkgsFor = forAllSystems (system:
        if (nixpkgs.legacyPackages.${system}.config.allowUnfree)
        then nixpkgs.legacyPackages.${system}
        else import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        }
      );
    in {
      homeModules = loadModulePaths "${self}/homeModules";
      packages = forAllSystems (system: import ./. { pkgs = pkgsFor.${system}; });
      releases = forAllSystems (system:
        let
          lib = nixpkgs.lib;
          pkgs = lib.filterAttrs (name: pkg:
            (pkg ? bundled) && (lib.isDerivation pkg.bundled)
          ) self.packages.${system};
        in
          lib.mapAttrs (name: pkg: pkg.bundled) pkgs
      );
      devShells = forAllSystems (system: {
        pacman = pkgsFor.${system}.mkShell {
          inputsFrom = [ self.packages.${system}.pacman ];
        };
      });
      # Matrix output for GitHub Actions to build all releases
      matrix = let
        lib = nixpkgs.lib;
        # Mapping of Nix systems to GitHub Actions runners
        runners = {
          "aarch64-darwin" = "macos-15";
          "x86_64-darwin" = "macos-15-intel";
        };
        jobs = lib.flatten (lib.mapAttrsToList
          (system: runner: lib.mapAttrsToList
            (app: drv: {
              inherit runner system app;
              inherit (drv) version;
              name = "${app}-${system}";
              command = "nix build .#releases.${system}.${app}";
            })
            self.releases.${system}
          )
          runners
        );
      in { include = jobs; };
    };
}
