{ pkgs, ... }:
{
  # go.mod: go 1.24.0 (minimum). Track pkgs.go for whatever version nixos-26.05
  # ships, keeping devenv's Go tooling (gopls etc.) in sync with the compiler.
  languages.go = {
    enable = true;
    package = pkgs.go;
  };

  packages = with pkgs; [ git gnumake oci-cli ];
}
