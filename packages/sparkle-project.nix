{
  lib,
  stdenv,
  fetchzip,
}:

stdenv.mkDerivation (final: {
  pname = "sparkle-binary";
  version = "2.8.1";

  src = fetchzip {
    url = "https://github.com/sparkle-project/Sparkle/releases/download/${final.version}/Sparkle-${final.version}.tar.xz";
    sha256 = "sha256-oZSkPcAnjCeNlfkQ8gyilPEIRqjnYWU/jBYrLonp7c0=";
    stripRoot = false;
  };

  installPhase = ''
    mkdir -p $out/Library/Frameworks
    cp -r Sparkle.framework $out/Library/Frameworks/
  '';

  meta = {
    description = "Pre-built binary of Sparkle, a software update framework for macOS";
    homepage = "https://sparkle-project.org/";
    license = lib.licenses.mit;
    platforms = lib.platforms.darwin;
  };
})
