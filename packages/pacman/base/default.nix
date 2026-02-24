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

  parseSection = section:
    let
      lines = lib.map lib.trim (lib.splitString "\n" section);
      version = lib.removePrefix "# v" (lib.findFirst (lib.hasPrefix "# v") "" lines);
      changes = lib.map
        (lib.removePrefix "* ")
        (lib.filter (lib.hasPrefix "* ") lines);
    in
      lib.optionalAttrs (version != "") {
        inherit version changes;
      };

  parseChangelog = content:
    let
      sections = lib.map
        (section: parseSection (lib.trim section))
        (lib.splitString "___" content);
    in
      lib.filter (x: x != {}) sections;

  changelog = parseChangelog (builtins.readFile ./src/docs/CHANGELOG.md);
in buildGoModule (final: {
  pname = "pacman";
  version = (builtins.elemAt changelog 0).version;

  meta = {
    description = "Rule-based HTTP proxy server.";
    homepage = "https://github.com/gilliginsisland/flakes";
    platforms = lib.platforms.all;
    mainProgram = "pacman";
    changes = (builtins.elemAt changelog 0).changes;
  };

  nativeBuildInputs = [ pkg-config ];
  buildInputs = [ openconnect' apple-sdk_15 (darwinMinVersionHook "14.4") ];

  ldflags = [
    "-X github.com/gilliginsisland/pacman/internal/version.Version=${final.version}"
  ];

  env = {
    CGO_ENABLED = "1";
  };

  vendorHash = null;

  src = lib.cleanSource ./src;
})
