{ lib, buildGoModule }:

let
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

  changelog = lib.filter (x: x != {}) (lib.map
    (section: parseSection (lib.trim section))
    (lib.splitString "___" (builtins.readFile ./CHANGELOG.md)));
  release = builtins.elemAt changelog 0;
in buildGoModule rec {
  pname = "git-remote-dropbox";
  version = release.version;

  src = lib.cleanSource ./.;
  vendorHash = null;
  subPackages = [ "." ];

  ldflags = [
    "-s"
    "-w"
    "-X github.com/gilliginsisland/flakes/git-remote-dropbox/cmd.Version=${version}"
  ];

  meta = {
    description = "Use a Dropbox folder as a Git remote";
    homepage = "https://github.com/gilliginsisland/flakes/tree/develop/packages/git-remote-dropbox";
    license = lib.licenses.mit;
    platforms = lib.platforms.unix;
    mainProgram = "git-remote-dropbox";
    changes = release.changes;
  };
}
