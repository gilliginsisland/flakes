{
  pacman,
  openconnect-slim,
  macdylibbundler,
  runCommand,
}:

let
  pacman-slim = pacman.override {
    openconnect = openconnect-slim;
  };
in

runCommand "pacman-bundled" {
  version = pacman.version;
  nativeBuildInputs = [ macdylibbundler ];
  buildInputs = [ pacman-slim ];
} ''
  app="$out/Applications/PACman.app"
  mkdir -p "$app"
  cp -Tr '${pacman-slim}/Applications/PACman.app' "$app"
  chmod -R u+w "$app"

  # Bundle dynamic libraries into Frameworks
  echo "Embedding dynamic libraries..."
  dylibbundler \
    -b -cd -ns \
    -x "$app/Contents/MacOS/PACman" \
    -d "$app/Contents/libs" \
    -p @executable_path/../libs
''
