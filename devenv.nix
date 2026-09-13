{ pkgs, ... }:
{
  # go.mod: go 1.24.0 (minimum). nixpkgs-unstable no longer ships go_1_24 and
  # devenv's Go tooling (gopls etc.) needs >= 1.26, so track pkgs.go.
  languages.go = {
    enable = true;
    package = pkgs.go;
  };

  packages = with pkgs; [ git gnumake oci-cli ];
}
