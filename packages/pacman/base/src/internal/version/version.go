package version

// Version is the display version, set via -ldflags at build time.
// Default to "dev" for non-Nix/manual builds.
var Version = "dev"
