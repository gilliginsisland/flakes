{ lib, buildGoModule }:

buildGoModule rec {
  name = "launch_socket_server";

  # no vendor folder
  vendorHash = null;

  src = lib.cleanSource ./.;
}
