{
  description = "glow - development and distribution";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default";
  };

  outputs = inputs @ {...}:
    inputs.flake-parts.lib.mkFlake {inherit inputs;} {
      debug = true;

      systems = [
        "x86_64-linux"
        "aarch-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      perSystem = {pkgs, ...}: {
        formatter = pkgs.alejandra;
        packages.default = pkgs.callPackage ./. {inherit pkgs;};

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go_1_26
          ];

          shellHook =
            # sh
            ''
              export CGO_ENABLED=0
            '';
        };
      };
    };
}
