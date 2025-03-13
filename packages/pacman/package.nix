{ lib, buildGoModule }:

buildGoModule {
  pname = "pacman";
  version = "2.0";

  meta = with lib; {
    description = "Rule-based HTTP proxy server.";
    homepage = "https://github.com/gilliginsisland/flakes";
    platforms = platforms.all;
    mainProgram = "pacman";
  };

  vendorHash = null;

  src = lib.cleanSource ./.;
}
