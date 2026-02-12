{
  lib,
  stdenv,
  librsvg,
  runCommand,
  callPackage,
}:

let
  pacman = callPackage ../base {};

  iconutil = stdenv.mkDerivation {
    name = "iconutil";

    phases = [ "installPhase" ];
    installPhase = ''
      mkdir -p $out/bin
      ln -s /usr/bin/iconutil $out/bin/iconutil
    '';

    meta = {
      description = "Wrapper exposing macOS native iconutil command";
      platforms = lib.platforms.darwin;
    };
  };

  codesign = stdenv.mkDerivation {
    name = "codesign";

    phases = [ "installPhase" ];
    installPhase = ''
      mkdir -p $out/bin
      ln -s /usr/bin/codesign $out/bin/codesign
    '';

    meta = {
      description = "Wrapper exposing macOS native codesign command";
      platforms = lib.platforms.darwin;
    };
  };

  resources = lib.cleanSource ./resources;

  pacman-app = runCommand "pacman-app" {
    inherit (pacman) version;
    nativeBuildInputs = [ librsvg iconutil codesign ];
    buildInputs = [ pacman ];
    meta = pacman.meta // {
      mainProgram = "pacman";
      platforms = lib.platforms.darwin;
    };
    passthru = {
      bundled = callPackage ../bundled { inherit pacman-app; };
    };
  } ''
    # Prepare app bundle
    app=$out/Applications/PACman.app
    mkdir -p "$app"/Contents/{MacOS,Resources}
    cp "${pacman}/bin/pacman" "$app/Contents/MacOS/PACman"

    # Generate .icns from icon.png
    input=${resources}/icon.svg
    mkdir -p icon.iconset
    rsvg-convert -w 16 -h 16     $input -o icon.iconset/icon_16x16.png
    rsvg-convert -w 32 -h 32     $input -o icon.iconset/icon_16x16@2x.png
    rsvg-convert -w 32 -h 32     $input -o icon.iconset/icon_32x32.png
    rsvg-convert -w 64 -h 64     $input -o icon.iconset/icon_32x32@2x.png
    rsvg-convert -w 128 -h 128   $input -o icon.iconset/icon_128x128.png
    rsvg-convert -w 256 -h 256   $input -o icon.iconset/icon_128x128@2x.png
    rsvg-convert -w 256 -h 256   $input -o icon.iconset/icon_256x256.png
    rsvg-convert -w 512 -h 512   $input -o icon.iconset/icon_256x256@2x.png
    rsvg-convert -w 512 -h 512   $input -o icon.iconset/icon_512x512.png
    rsvg-convert -w 1024 -h 1024 $input -o icon.iconset/icon_512x512@2x.png
    iconutil -c icns icon.iconset

    # Copy assets
    cp icon.icns "$app/Contents/Resources/icon.icns"
    cp ${resources}/menuicon.pdf "$app/Contents/Resources/menuicon.pdf"
    cp ${resources}/Info.plist "$app/Contents/Info.plist"

    codesign -s - --force --deep --timestamp --entitlements ${resources}/entitlements.plist "$app"

    mkdir -p "$out/bin"
    ln -s "$app/Contents/MacOS/PACman" "$out/bin/pacman"
  '';
in
  pacman-app
