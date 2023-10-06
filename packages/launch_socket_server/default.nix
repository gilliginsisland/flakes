{ stdenv, lib, buildGoPackage, fetchFromGitHub }:

buildGoPackage rec {
  name = "launch_socket_server";
  goPackagePath = name;

  # no vendor folder
  vendorSha256 = lib.fakeSha256;

  src = fetchFromGitHub {
    owner = "mistydemeo";
    repo = name;
    rev = "v2.0.0";
    sha256 = "sha256-y3pHxB+IiIXS60+4oGCWw4bTY4dJZXwYjWxFg2MuSMA=";
  };

  preBuild = ''
    mv go/src/${goPackagePath}/src/launch go/src/launch
    mv go/src/${goPackagePath} src
    mv src/src go/src/${goPackagePath}
  '';
}
