{ lib, buildGoModule }:

buildGoModule rec {
  name = "launch_socket_server";

  meta = with lib; {
    description = "A program that adds support for launchd activation to any process.";
    homepage = "https://github.com/gilliginsisland/flakes";
    platforms = platforms.all;
    mainProgram = "launch_socket_server";
  };

  # no vendor folder
  vendorHash = null;

  src = lib.cleanSource ./.;
}
