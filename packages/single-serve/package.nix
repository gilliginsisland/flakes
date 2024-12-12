{ lib, buildGoModule }:

buildGoModule {
	pname = "single-serve";
	version = "1.0";

	meta = with lib; {
		description = "Simple HTTP Server for serving a single file over stdio.";
		homepage = "https://github.com/gilliginsisland/flakes";
		platforms = platforms.all;
		mainProgram = "single-serve";
	};

	# no vendor folder
	vendorHash = null;

	src = lib.cleanSource ./.;
}
