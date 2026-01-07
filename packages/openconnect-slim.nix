{
  openconnect,
}:

(openconnect.override {
  vpnc-scripts = "/etc/vpnc/vpnc-script";
  stoken = null;
  useOpenSSL = true;
}).overrideAttrs (prev: {
  configureFlags = prev.configureFlags ++ [
    "--without-libpcsclite"
    "--without-stoken"
  ];
})
