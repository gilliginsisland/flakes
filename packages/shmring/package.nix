{ lib, stdenv }:

let
  library = "libshmring${stdenv.hostPlatform.extensions.sharedLibrary}";
in stdenv.mkDerivation (final: {
  pname = "shmring";
  version = "0.1.0";

  outputs = [ "out" "dev" ];
  src = lib.cleanSource ./src;
  dontConfigure = true;

  buildPhase = ''
    runHook preBuild
    $CC -std=c11 -O2 -fPIC -c shm_ring.c -o shm_ring.o
    $CC -std=c11 -O2 -fPIC -c shm_bridge.c -o shm_bridge.o
    $CC ${if stdenv.hostPlatform.isDarwin then
      "-dynamiclib -Wl,-install_name,$out/lib/${library}"
    else
      "-shared -Wl,-soname,${library}"
    } shm_ring.o shm_bridge.o -o ${library}
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    mkdir -p "$out/lib" "$dev/include" "$dev/lib/pkgconfig"
    cp ${library} "$out/lib/"
    cp shm_ring.h shm_bridge.h "$dev/include/"
    substitute shmring.pc.in "$dev/lib/pkgconfig/shmring.pc" \
      --replace-fail '@out@' "$out" \
      --replace-fail '@dev@' "$dev" \
      --replace-fail '@version@' '${final.version}'
    runHook postInstall
  '';

  meta = {
    description = "Shared-memory rings and bidirectional record transport";
    platforms = lib.platforms.unix;
  };
})
