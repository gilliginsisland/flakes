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
  rcodesign,
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

  updater-bundle = stdenv.mkDerivation {
    name = "updater-bundle";
    src = lib.fileset.toSource {
      root = ./src;
      fileset = lib.fileset.unions [
        ./src/updater_plugin.m
        ./src/updater_plugin-Info.plist
      ];
    };
    nativeBuildInputs = [ cctools ];
    buildInputs = [ sparkle-slim ];
    buildPhase = ''
      cp ${../../base/src/internal/app/plugin/PACmanPlugin.h} PACmanPlugin.h
      clang -bundle updater_plugin.m -o updater_plugin \
        -fobjc-arc \
        -framework Cocoa \
        -framework Sparkle \
        -F"${sparkle-slim}/Library/Frameworks"
    '';
    installPhase = ''
      mkdir -p "$out/updater_plugin.bundle/Contents/MacOS"
      cp updater_plugin "$out/updater_plugin.bundle/Contents/MacOS/"
      cp updater_plugin-Info.plist "$out/updater_plugin.bundle/Contents/Info.plist"
    '';
  };

  attrs = {
    inherit (pacman) pname version meta;
    passthru = { inherit (pacman) releaseVersion; };
    nativeBuildInputs = [ macdylibbundler cctools insert-dylib rcodesign ];
    buildInputs = [ pacman openssl.out ossl-conf-sidecar updater-bundle ];
  };
in

runCommand "pacman-bundled" attrs ''
  app="$out/Applications/PACman.app"
  mkdir -p "$app/Contents/"{Frameworks,PlugIns,Resources}
  cp -Tr "${sparkle-slim}/Library/Frameworks/Sparkle.framework" "$app/Contents/Frameworks/Sparkle.framework"
  cp -Tr "${openssl.out}/etc/ssl" "$app/Contents/Resources/ssl"
  cp -Tr "${openssl.out}/lib/engines-3" "$app/Contents/PlugIns/engines-3"
  cp -Tr "${openssl.out}/lib/ossl-modules" "$app/Contents/PlugIns/ossl-modules"
  cp -Tr "${updater-bundle}/updater_plugin.bundle" "$app/Contents/PlugIns/updater_plugin.bundle"
  cp -Tr "${pacman}/Applications/PACman.app" "$app"
  chmod -R u+w "$app"

  insert_dylib \
    "${ossl-conf-sidecar}/lib/libossl_conf_sidecar.dylib" \
    "$app/Contents/MacOS/PACman" \
    --inplace --overwrite \
    --all-yes --no-strip-codesig

  /usr/libexec/PlistBuddy -c \
    "Add :SUFeedURL string 'https://gilliginsisland.github.io/flakes/appcasts/${attrs.pname}-${stdenv.hostPlatform.system}-appcast.xml'" \
    "$app/Contents/Info.plist"

  dylibs=(
    "$app/Contents/MacOS/PACman"
    "$app/Contents/PlugIns/updater_plugin.bundle/Contents/MacOS/updater_plugin"
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

  rcodesign sign \
    --timestamp-url none \
    "$app"
''
