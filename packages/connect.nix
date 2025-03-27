{
  lib,
  stdenv,
  fetchFromGitHub,
  gcc
}:

stdenv.mkDerivation rec {
  pname = "connect";
  version = "1.105";

  src = fetchFromGitHub {
    owner = "gotoh";
    repo = "ssh-connect";
    rev = version;
    sha256 = "sha256-pj7DadXJEsgm3WZOVYAQBIk3qGDKPu4iONJYc0Et/Sc=";
  };

  nativeBuildInputs = [ gcc ];

  buildPhase = ''
    make
  '';

  installPhase = ''
    mkdir -p $out/bin
    cp connect $out/bin/connect
  '';

  meta = with lib; {
    description = "HTTP / SOCKS Proxy Command";
    homepage = "https://github.com/gotoh/ssh-connect";
    mainProgram = "connect";
    platforms = platforms.all;
  };
}
