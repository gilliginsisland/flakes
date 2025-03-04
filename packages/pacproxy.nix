{
  lib,
  buildGoModule,
  fetchFromGitHub,
}:

buildGoModule {
  pname = "pacproxy";
  version = "2.0.5";

  doCheck = false;

  src = fetchFromGitHub {
    owner = "edolphin-ydf";
    repo = "pacproxy";
    rev = "master";
    hash = "sha256-L+0N9WoU3ZPCIm6UlBl+cdeR1a9sXpbYOZvIpy9qxQk=";
  };

  vendorHash = "sha256-CufFcylQ5OuP0Oh6RBvQ4uWWadFQdLIl9RVGuQ2Qsbk=";

  meta = with lib; {
    description = "No-frills local HTTP proxy server powered by a proxy auto-config (PAC) file";
    homepage = "https://github.com/edolphin-ydf/pacproxy";
    changelog = "https://github.com/edolphin-ydf/pacproxy/commits/master";
    license = licenses.asl20;
    mainProgram = "pacproxy";
  };
}
