{
  lib,
  buildGoModule,
  fetchgit,
  pkg-config,
  openconnect,
  macdylibbundler,
  writeTextFile,
  stdenv,
}:

let
  openconnect' = openconnect.overrideAttrs (prev: {
    version = "9.12.1";

    src = fetchgit {
      url = "git://git.infradead.org/users/dwmw2/openconnect.git";
      rev = "f17fe20d337b400b476a73326de642a9f63b59c8";
      sha256 = "sha256-OBEojqOf7cmGtDa9ToPaJUHrmBhq19/CyZ5agbP7WUw=";
    };

    patches = (prev.patches or []) ++ (
      let
        patchDir = builtins.path {
          name = "patches";
          path = ./patches;
          filter = path: type: lib.hasSuffix ".patch" path;
        };
      in
        builtins.map
          (name: "${patchDir}/${name}")
          (builtins.attrNames (builtins.readDir patchDir))
    );

    # Remove the old vpnc-script setting
    configureFlags = builtins.filter
      (flag: !lib.hasPrefix "--with-vpnc-script=" flag)
      (prev.configureFlags or []) ++ [
        "--with-vpnc-script=/bin/true"
      ];
  });
in

buildGoModule {
  pname = "pacman";
  version = "2.0";

  meta = with lib; {
    description = "Rule-based HTTP proxy server.";
    homepage = "https://github.com/gilliginsisland/flakes";
    platforms = platforms.all;
    mainProgram = "pacman";
  };

  nativeBuildInputs = [ pkg-config macdylibbundler ];
  buildInputs = [ openconnect' ];

  # Ensure cgo picks up the correct .pc with internal header path
  env = {
    PKG_CONFIG_PATH = "${openconnect'}/lib/pkgconfig";
    CGO_ENABLED = "1";
  };

  vendorHash = null;

  src = lib.cleanSource ./.;

  installPhase = ''
    app=$out/Applications/Pacman.app
    mkdir -p "$app"/Contents/{MacOS,Resources,lib}
    cp "$GOPATH/bin/pacman" "$app/Contents/MacOS/Pacman"

    # Copy Info.plist
    cp "${./Info.plist}" "$app/Contents/Info.plist"

    # Bundle dynamic libraries into Frameworks
    "${lib.getBin macdylibbundler}/bin/dylibbundler" \
      -b \
      -x "$app/Contents/MacOS/Pacman" \
      -d "$app/Contents/lib" \
      -p @executable_path/../lib

    mkdir -p "$out/bin"
    ln -s "$app/Contents/MacOS/Pacman" "$out/bin/pacman"
  '';
}
