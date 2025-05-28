{
  lib,
  buildGoModule,
  fetchgit,
  pkg-config,
  openconnect,
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

  nativeBuildInputs = [ pkg-config ];
  buildInputs = [ openconnect' ];

  # Ensure cgo picks up the correct .pc with internal header path
  env = {
    PKG_CONFIG_PATH = "${openconnect'}/lib/pkgconfig";
    CGO_ENABLED = "1";
  };

  vendorHash = null;

  src = lib.cleanSource ./.;

  preBuild = lib.optionalString stdenv.isDarwin ''
    arch=$(uname -m)
    case "$arch" in
      arm64)  suffix=darwin_arm64 ;;
      x86_64) suffix=darwin_amd64 ;;
      *)      echo "Unsupported architecture: $arch" >&2; exit 1 ;;
    esac

    echo "Embedding Info.plist for $suffix"

    # Minimal empty object file
    echo | clang -x assembler -c -o dummy.o -

    # Inject Info.plist as a section
    ld -r -sectcreate __TEXT __info_plist Info.plist -o info_plist_''${suffix}.syso -arch $arch dummy.o

    mv info_plist_''${suffix}.syso cmd/pacman/info_plist_''${suffix}.syso
  '';
}
