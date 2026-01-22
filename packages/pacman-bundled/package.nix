{
  lib,
  stdenv,
  pacman-app,
  openssl,
  libiconv,
  darwin,
  macdylibbundler,
  runCommand,
  cctools,
  insert-dylib,
}:

let
  ossl-conf-sidecar = stdenv.mkDerivation {
    name = "ossl-conf-sidecar";
    src = ./.;
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

  codesign = stdenv.mkDerivation {
    name = "codesign";

    phases = [ "installPhase" ];
    installPhase = ''
      mkdir -p $out/bin
      ln -s /usr/bin/codesign $out/bin/codesign
    '';

    meta = with lib; {
      description = "Wrapper exposing macOS native codesign command";
      platforms = platforms.darwin;
    };
  };
in

runCommand "pacman-bundled" {
  version = pacman-app.version;
  nativeBuildInputs = [ macdylibbundler cctools insert-dylib codesign ];
  buildInputs = [ pacman-app openssl.out ossl-conf-sidecar ];
} ''
  app="$out/Applications/PACman.app"
  mkdir -p "$app/Contents/"{Frameworks,PlugIns,Resources}
  cp -Tr "${openssl.out}/etc/ssl" "$app/Contents/Resources/ssl"
  cp -Tr "${openssl.out}/lib/engines-3" "$app/Contents/PlugIns/engines-3"
  cp -Tr "${openssl.out}/lib/ossl-modules" "$app/Contents/PlugIns/ossl-modules"
  cp -Tr "${pacman-app}/Applications/PACman.app" "$app"
  chmod -R u+w "$app"

  insert_dylib \
    "${ossl-conf-sidecar}/lib/libossl_conf_sidecar.dylib" \
    "$app/Contents/MacOS/PACman" --inplace --overwrite

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

  codesign -s - --force --deep --timestamp --entitlements ${../pacman-app/entitlements.plist} "$app"
''
