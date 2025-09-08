{
  lib,
  buildGoModule,
  pkg-config,
  openconnect,
  macdylibbundler,
  stdenv,
  librsvg,
  apple-sdk_15
}:

let
  iconutil = stdenv.mkDerivation {
    name = "iconutil";

    phases = [ "installPhase" ];
    installPhase = ''
      mkdir -p $out/bin
      ln -s /usr/bin/iconutil $out/bin/iconutil
    '';

    meta = with lib; {
      description = "Wrapper exposing macOS native iconutil command";
      platforms = platforms.darwin;
    };
  };
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

  nativeBuildInputs = [ pkg-config macdylibbundler librsvg iconutil ];
  buildInputs = [ apple-sdk_15 openconnect.dev ];

  # Ensure cgo picks up the correct .pc with internal header path
  env = {
    CGO_ENABLED = "1";
  };

  vendorHash = null;

  src = lib.cleanSource ./.;

  installPhase = ''
    app=$out/Applications/PACman.app
    mkdir -p "$app"/Contents/{MacOS,Resources,lib}
    cp "$GOPATH/bin/pacman" "$app/Contents/MacOS/PACman"

    # Generate .icns from icon.png
    mkdir -p icon.iconset
    rsvg-convert -w 16 -h 16     icon.svg -o icon.iconset/icon_16x16.png
    rsvg-convert -w 32 -h 32     icon.svg -o icon.iconset/icon_16x16@2x.png
    rsvg-convert -w 32 -h 32     icon.svg -o icon.iconset/icon_32x32.png
    rsvg-convert -w 64 -h 64     icon.svg -o icon.iconset/icon_32x32@2x.png
    rsvg-convert -w 128 -h 128   icon.svg -o icon.iconset/icon_128x128.png
    rsvg-convert -w 256 -h 256   icon.svg -o icon.iconset/icon_128x128@2x.png
    rsvg-convert -w 256 -h 256   icon.svg -o icon.iconset/icon_256x256.png
    rsvg-convert -w 512 -h 512   icon.svg -o icon.iconset/icon_256x256@2x.png
    rsvg-convert -w 512 -h 512   icon.svg -o icon.iconset/icon_512x512.png
    rsvg-convert -w 1024 -h 1024 icon.svg -o icon.iconset/icon_512x512@2x.png
    iconutil -c icns icon.iconset

    # Add the icon to the app bundle
    cp icon.icns "$app/Contents/Resources/icon.icns"
    cp menuicon.pdf "$app/Contents/Resources/menuicon.pdf"

    # Copy Info.plist
    cp Info.plist "$app/Contents/Info.plist"

    # Bundle dynamic libraries into Frameworks
    "${lib.getBin macdylibbundler}/bin/dylibbundler" \
      -b \
      -x "$app/Contents/MacOS/PACman" \
      -d "$app/Contents/lib" \
      -p @executable_path/../lib

    mkdir -p "$out/bin"
    echo '#!/bin/sh' > "$out/bin/pacman"
    echo exec "'$app/Contents/MacOS/PACman'" '"$@"' >> "$out/bin/pacman"
    chmod +x "$out/bin/pacman"
  '';
}
