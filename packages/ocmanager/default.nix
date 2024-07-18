{
  stdenv,
  bash,
  coreutils,
  envsubst,
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
  buildInputs = [
    bash coreutils netcat terminal-notifier
    openconnect ocproxy xmlstarlet gnused curl
    security osascript envsubst
  ];

  coreutils_root = coreutils;
  netcat_root = netcat;
  terminal_notifier_root = terminal-notifier;
  ocproxy_root = ocproxy;
  openconnect_root = openconnect;
  xmlstarlet_root = xmlstarlet;
  gnused_root = gnused;
  curl_root = curl;
  security_root = security;
  osascript_root = osascript;
  envsubst_root = envsubst;

  installPhase = ''
    cp -p -R src $out
  '';

  postFixup = ''
    while IFS= read -r -d $'\0' f; do
      substituteAllInPlace "$f"
    done < <(find "$out" -type f -perm -0100 -print0)
  '';
}
