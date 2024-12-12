{ lib, buildGoModule }:

buildGoModule {
  pname = "launch-socket-server";
  version = "2.0";

  meta = with lib; {
    description = "A program that adds support for launchd activation to any process.";
    homepage = "https://github.com/gilliginsisland/flakes";
    platforms = platforms.all;
    mainProgram = "launch-socket-server";
  };

  # no vendor folder
  vendorHash = null;

  src = lib.cleanSource ./.;
}
