{
  lib,
  buildGoModule,
  pkg-config,
  openconnect_openssl,
  apple-sdk_15,
  darwinMinVersionHook,
}:

let
  openconnect = (openconnect_openssl.override {
    vpnc-scripts = "/etc/vpnc/vpnc-script";
    stoken = null;
  }).overrideAttrs (prev: {
    patches = (prev.patches or []) ++ (
      let
        patches = builtins.path {
          name = "patches";
          path = ./patches;
          filter = path: type: lib.hasSuffix ".patch" path;
          # sha256 = lib.fakeHash;
        };
      in builtins.map
        (name: "${patches}/${name}")
        (builtins.attrNames (builtins.readDir patches))
    );

    configureFlags = prev.configureFlags ++ [
      "--without-libpcsclite"
      "--without-stoken"
    ];
  });
in buildGoModule {
  pname = "pacman";
  version = "2.0";

  meta = with lib; {
    description = "Rule-based HTTP proxy server.";
    homepage = "https://github.com/gilliginsisland/flakes";
    platforms = platforms.all;
    mainProgram = "pacman";
  };

  nativeBuildInputs = [ pkg-config ];
  buildInputs = [ openconnect apple-sdk_15 (darwinMinVersionHook "14.4") ];

  env = {
    CGO_ENABLED = "1";
  };

  vendorHash = null;

  src = lib.cleanSource ./.;
}
