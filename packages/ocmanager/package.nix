{
  lib,
  stdenv,
  bash,
  coreutils,
  envsubst,
  netcat,
  terminal-notifier,
  openconnect,
  openssl,
  xmlstarlet,
  gnused,
  curl,
  security,
  osascript
}:

let
  inherit(builtins)
    attrValues
    replaceStrings
    concatStringsSep;

  inherit(lib)
    platforms
    mapAttrsToList
    toUpper
    traceVal;

  inputs = {
    inherit
      coreutils
      netcat
      terminal-notifier
      openconnect
      xmlstarlet
      gnused
      curl
      security
      osascript
      envsubst
      openssl;
  };

  formatEnvVar = name: replaceStrings ["-"] ["_"] (toUpper name);
  mkSubstArg = name: pkg: "--subst-var-by '${formatEnvVar name}' '${pkg}'";
  substitutions = concatStringsSep " " (mapAttrsToList mkSubstArg inputs);
in stdenv.mkDerivation rec {
  name = "ocmanager";

  meta = {
    description = "A program that manages openconnect VPNs on demand.";
    homepage = "https://github.com/gilliginsisland/flakes";
    platforms = platforms.all;
    mainProgram = "ocmanager";
  };

  src = ./.;
  buildInputs = [ bash ] ++ attrValues inputs;

  installPhase = ''
    cp -p -R src $out
  '';

  postFixup = ''
    while IFS= read -r -d $'\0' f; do
      substituteInPlace "$f" ${substitutions} --subst-var-by "SELF" "$out"
    done < <(find "$out" -type f -perm -0100 -print0)
  '';
}
