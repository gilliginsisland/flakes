{
  stdenv,
  callPackage,
}:

if stdenv.isDarwin then callPackage ./darwin {} else callPackage ./base {}
