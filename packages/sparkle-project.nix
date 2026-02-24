{
  lib,
  stdenv,
  fetchzip,
}:

stdenv.mkDerivation (final: {
  pname = "sparkle-binary";
  version = "2.9.0";

  src = fetchzip {
    url = "https://github.com/sparkle-project/Sparkle/releases/download/${final.version}/Sparkle-${final.version}.tar.xz";
    sha256 = "sha256-1P9l2K1ODkFs4BF4xUVFQMUvEvOWa6g+fdVeboCAhAI=";
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
