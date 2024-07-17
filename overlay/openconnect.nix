final: prev:

if prev.stdenv.isDarwin then prev.openconnect.override {
  pcsclite = final.darwin.apple_sdk.frameworks.PCSC;
} else prev.openconnect
