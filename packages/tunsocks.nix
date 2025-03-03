{
  lib,
  stdenv,
  fetchFromGitHub,
  pkg-config,
  autoreconfHook
}:

stdenv.mkDerivation rec {
  pname = "tunsocks";
  version = "2023.6.22";

  meta = with lib; {
    description = "User-level IP forwarding, SOCKS proxy, and HTTP proxy for VPNs that provide tun-like interface.";
    homepage = "https://github.com/flavio/tunsocks";
    platforms = platforms.all;
    mainProgram = "tunsocks";
  };

  src = fetchFromGitHub {
    owner = "russdill";
    repo = "tunsocks";
    rev = "4e4ff8682053412145930b8daf2c55d357cf1e44";
    hash = "";
    fetchSubmodules = true;
  };

  nativeBuildInputs = [
    pkg-config
    autoreconfHook
  ];
}
