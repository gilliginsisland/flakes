{
  description = "Packages related to on demand AnyConnect VPNs";

  outputs = { self, nixpkgs }:
    let
      # System types to support.
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      # Helper function to generate an attrset '{ x86_64-linux = f "x86_64-linux"; ... }'.
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      # Helper function to generate the packages for a specific system
      genSystemPackages = (system: import ./. {
        pkgs = import nixpkgs { inherit system; };
      });
    in {
      packages = forAllSystems genSystemPackages;
    };
}
