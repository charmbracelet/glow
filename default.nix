{pkgs ? import <nixpkgs> {}, ...}:
pkgs.buildGoModule {
  pname = "glow";
  version = "2.1.2-pre";
  vendorHash = "sha256-o5Z2ABRw6v4wFXp+KxgdKQn5/Lk5LG73VTiDOA/kBIs=";

  src = builtins.path {
    path = ./.;
    name = "source";
  };
}
