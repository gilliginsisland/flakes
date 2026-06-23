{
  coreutils,
  file,
  findutils,
  gnutar,
  lib,
  nix,
  rcodesign,
  writeShellApplication,
}:

writeShellApplication {
  name = "flakesign";

  runtimeInputs = [
    coreutils
    file
    findutils
    gnutar
    nix
    rcodesign
  ];

  text = builtins.readFile ./flakesign.sh;

  meta = {
    description = "Build, sign, and archive flake app bundle outputs";
    mainProgram = "flakesign";
    platforms = lib.platforms.all;
  };
}
