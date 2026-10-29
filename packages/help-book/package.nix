{ lib, buildGoModule }:

buildGoModule {
  pname = "help-book";
  version = "1.0";

  src = lib.cleanSource ./.;
  vendorHash = null;

  meta = {
    description = "Generate a static HTML help book from Markdown";
    homepage = "https://github.com/gilliginsisland/flakes";
    license = lib.licenses.unlicense;
    platforms = lib.platforms.all;
    mainProgram = "help-book";
  };
}
