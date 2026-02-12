{
  lib,
  buildGoModule,
  pkg-config,
  openconnect_openssl,
  apple-sdk_15,
  darwinMinVersionHook,
}:

let
  openconnect' = (openconnect_openssl.override {
    vpnc-scripts = "/etc/vpnc/vpnc-script";
    stoken = null;
  }).overrideAttrs (prev: {
    patches = (prev.patches or []) ++ lib.filesystem.listFilesRecursive (lib.cleanSource ./src/patches);
    configureFlags = prev.configureFlags ++ [
      "--without-libpcsclite"
      "--without-stoken"
    ];
  });
in buildGoModule {
  pname = "pacman";
  version = "2.0";

  meta = {
    description = "Rule-based HTTP proxy server.";
    homepage = "https://github.com/gilliginsisland/flakes";
    platforms = lib.platforms.all;
    mainProgram = "pacman";
  };

  nativeBuildInputs = [ pkg-config ];
  buildInputs = [ openconnect' apple-sdk_15 (darwinMinVersionHook "14.4") ];

  env = {
    CGO_ENABLED = "1";
  };

  vendorHash = null;

  src = lib.cleanSource ./src;
}
