package version

// Version is set via -ldflags at build time.
// Default to "dev" for non-Nix/manual builds.
var Version = "dev"
