{
  lib,
  stdenv,
  pacman,
  openssl,
  libiconv,
  darwin,
  macdylibbundler,
  runCommand,
  cctools,
  insert-dylib,
  sparkle-project,
}:

let
  ossl-conf-sidecar = stdenv.mkDerivation {
    name = "ossl-conf-sidecar";
    src = lib.fileset.toSource {
      root = ./src;
      fileset = ./src/ossl_conf_sidecar.c;
    };
    nativeBuildInputs = [ cctools ];
    buildPhase = ''
      clang -dynamiclib ossl_conf_sidecar.c -o libossl_conf_sidecar.dylib \
        -install_name "$out/lib/libossl_conf_sidecar.dylib"
    '';
    installPhase = ''
      mkdir -p $out/lib
      cp libossl_conf_sidecar.dylib $out/lib/
    '';
  };

  sparkle-slim = runCommand "sparkle-slim" {
    nativeBuildInputs = [ cctools ];
    buildInputs = [ sparkle-project ];
  } ''
    mkdir -p $out/Library/Frameworks
    cp -Tr "${sparkle-project}/Library/Frameworks/Sparkle.framework" "$out/Library/Frameworks/Sparkle.framework"
    chmod -R u+w "$out"
    rm -rf "$out/Contents/Frameworks/Sparkle.framework/"{XPCServices,Versions/*/XPCServices}
    install_name_tool -id \
      "@executable_path/../Frameworks/Sparkle.framework/Versions/B/Sparkle" \
      "$out/Library/Frameworks/Sparkle.framework/Sparkle"
  '';

  sparkle-sidecar = stdenv.mkDerivation {
    name = "sparkle-sidecar";
    src = lib.fileset.toSource {
      root = ./src;
      fileset = ./src/sparkle_sidecar.m;
    };
    nativeBuildInputs = [ cctools ];
    buildInputs = [ sparkle-slim ];
    buildPhase = ''
      clang -dynamiclib sparkle_sidecar.m -o libsparkle_sidecar.dylib \
        -fobjc-arc \
        -framework Cocoa \
        -framework Sparkle \
        -F"${sparkle-slim}/Library/Frameworks" \
        -install_name "$out/lib/libsparkle_sidecar.dylib"
    '';
    installPhase = ''
      mkdir -p $out/lib
      cp libsparkle_sidecar.dylib $out/lib/
    '';
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

  attrs = {
    inherit (pacman) pname version meta;
    nativeBuildInputs = [ macdylibbundler cctools insert-dylib codesign ];
    buildInputs = [ pacman openssl.out ossl-conf-sidecar sparkle-sidecar ];
  };
in

runCommand "pacman-bundled" attrs ''
  app="$out/Applications/PACman.app"
  mkdir -p "$app/Contents/"{Frameworks,PlugIns,Resources}
  cp -Tr "${sparkle-slim}/Library/Frameworks/Sparkle.framework" "$app/Contents/Frameworks/Sparkle.framework"
  cp -Tr "${openssl.out}/etc/ssl" "$app/Contents/Resources/ssl"
  cp -Tr "${openssl.out}/lib/engines-3" "$app/Contents/PlugIns/engines-3"
  cp -Tr "${openssl.out}/lib/ossl-modules" "$app/Contents/PlugIns/ossl-modules"
  cp -Tr "${pacman}/Applications/PACman.app" "$app"
  chmod -R u+w "$app"

  insert_dylib \
    "${ossl-conf-sidecar}/lib/libossl_conf_sidecar.dylib" \
    "$app/Contents/MacOS/PACman" \
    --inplace --overwrite \
    --all-yes --no-strip-codesig

  insert_dylib \
    "${sparkle-sidecar}/lib/libsparkle_sidecar.dylib" \
    "$app/Contents/MacOS/PACman" \
    --inplace --overwrite \
    --all-yes --no-strip-codesig

  /usr/libexec/PlistBuddy -c \
    "Add :SUFeedURL string 'https://gilliginsisland.github.io/flakes/appcasts/${attrs.pname}-${stdenv.hostPlatform.system}-appcast.xml'" \
    "$app/Contents/Info.plist"

  dylibs=(
    "$app/Contents/MacOS/PACman"
    "$app/Contents/PlugIns/engines-3/"*.dylib
    "$app/Contents/PlugIns/ossl-modules/"*.dylib
  )

  bundler_args=()
  for file in "''${dylibs[@]}"; do
    bundler_args+=("-x" "$file")
  done

  # Bundle dynamic libraries into Frameworks
  echo "Embedding dynamic libraries..."
  dylibbundler \
    -b -cd -ns \
    -i "${libiconv}/lib" \
    -i "${darwin.libresolv}/lib" \
    "''${bundler_args[@]}" \
    -d "$app/Contents/Frameworks" \
    -p @executable_path/../Frameworks

  for file in "''${dylibs[@]}" $app/Contents/Frameworks/*.dylib; do
    install_name_tool -change "${libiconv}/lib/libiconv.2.dylib" "/usr/lib/libiconv.2.dylib" "$file"
    install_name_tool -change "${darwin.libresolv}/lib/libresolv.9.dylib" "/usr/lib/libresolv.9.dylib" "$file"
  done

  codesign -s - --force --deep --timestamp --preserve-metadata "$app"
''
