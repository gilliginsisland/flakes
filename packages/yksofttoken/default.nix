{ stdenv, lib, fetchFromGitHub, libyubikey, gcc }:

stdenv.mkDerivation {
  name = "yksofttoken";

  src = fetchFromGitHub {
    owner = "arr2036";
    repo = "yksofttoken";
    rev = "v0.0.4";
    sha256 = "sha256-a4E21PRqfQRJx4WbhQWdsrSrkPwE2CgFCa3UAsqmNyA=";
  };

  buildInputs = [ gcc libyubikey ];

  buildPhase = ''
    gcc -g3 -Wall -I"${lib.getLib libyubikey}/include" -L"${lib.getLib libyubikey}/lib" -o ./yksoft -lyubikey "$src/yksoft.c"
  '';
  installPhase = ''
    mkdir -p "$out/bin"
    cp ./yksoft "$out/bin/"
  '';
}
