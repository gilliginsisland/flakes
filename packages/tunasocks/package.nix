{
  lib,
  buildGoModule,
}:

buildGoModule {
  pname = "tunasocks";
  version = "0.9";

  meta = with lib; {
    description = "OKE k8s bastion proxy server";
    homepage = "https://github.com/gilliginsisland/flakes/packages/tunasocks";
    platforms = platforms.all;
    mainProgram = "tunasocks";
  };

  env = {
    CGO_ENABLED = "1";
  };

  vendorHash = null;

  src = lib.cleanSource ./.;
}
