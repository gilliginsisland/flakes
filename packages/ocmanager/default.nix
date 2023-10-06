{
  stdenv,
  makeBinaryWrapper,
  lib,
  bash,
  coreutils,
  netcat,
  terminal-notifier,
  openconnect,
  ocproxy,
  xmlstarlet,
  gnused,
  curl,
  security,
  osascript
}:

stdenv.mkDerivation rec {
  name = "ocmanager";
  src = ./.;
  nativeBuildInputs = [ makeBinaryWrapper ];
  buildInputs = [
    bash coreutils netcat terminal-notifier
    openconnect ocproxy xmlstarlet gnused curl
    security osascript
  ];
  installPhase = ''
    cp -p -R src $out
  '';
  postFixup = ''
    while IFS= read -r -d $'\0' f; do
      wrapProgram "$f" \
        --set PATH ${lib.makeBinPath buildInputs}
    done < <(find "$out" -type f -perm -0100 -print0)
  '';
}
