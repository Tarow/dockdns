{
  description = "A Nix-flake-based Starforge development environment";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-24.11";
  };
  outputs = {nixpkgs, ...}: let
    systems = [
      "aarch64-linux"
      "i686-linux"
      "x86_64-linux"
      "aarch64-darwin"
      "x86_64-darwin"
    ];
    forAllSystems = nixpkgs.lib.genAttrs systems;
  in {
    devShells = forAllSystems (system: let
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      default = pkgs.mkShell {
        packages = with pkgs; [
          go
          golangci-lint
          air
          gopls
          gotools
          delve
          templ
        ];
      };
    });

    packages =
      forAllSystems
      (system: let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = nixpkgs.lib;
      in rec {
        default = starforge;

        starforge = pkgs.buildGoModule {
          name = "starforge";
          buildInputs = nixpkgs.lib.lists.optionals pkgs.stdenv.isDarwin [pkgs.darwin.apple_sdk.frameworks.AppKit];
          src = lib.fileset.toSource rec {
            root = ./.;
            fileset = lib.fileset.unions [
              ./go.mod
              ./go.sum
              ./staticpage
              (lib.fileset.fileFilter (file: file.hasExt "go") root)
              (lib.fileset.fileFilter (file: file.hasExt "yaml") root)
            ];
          };
          vendorHash = "sha256-m8NfBSgIkPRMRAXf5eTduZWQQ4gHJrRVWqgWPQrm42g=";
          meta.mainProgram = "starforge";
        };

        starforge-docker = pkgs.dockerTools.buildImage {
          name = "starforge";
          config = {
            Cmd = ["${lib.getExe starforge}"];
          };
        };
      });
  };
}
