{
  lib,
  fetchFromGitHub,
  ocproxy
}:

ocproxy.overrideAttrs (prev: rec {
  version = "2.0";

  src = fetchFromGitHub {
    owner = "gilliginsisland";
    repo = "ocproxy";
    rev = "v${version}";
    sha256 = "sha256-PJJ7qxI1Ax+rQkicU5J4HLsBsYp8oPY6sACzXgnFpss=";
  };

  meta = prev.meta // {
    mainProgram = "ocproxy";
  };
})
