{ lib, ... }:

with lib;

{
  imports = [
    ../pacman
    (mkRenamedOptionModule [ "programs" "proxypac" ] [ "programs" "pacman" ])
  ];
}
