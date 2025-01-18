{
  description = "DockDNS";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-24.11";
    nixpkgs-unstable.url = "github:nixos/nixpkgs/nixos-unstable";
  };
  outputs = {
    self,
    nixpkgs,
    nixpkgs-unstable,
    ...
  }: let
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
      unstable = nixpkgs-unstable.legacyPackages.${system};
    in {
      default = pkgs.mkShell {
        packages = with pkgs;
          [
            go
            golangci-lint
            air
            gopls
            gotools
            delve
          ]
          ++ (with unstable; [
            templ
          ]);
      };
    });

    packages =
      forAllSystems
      (system: let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = nixpkgs.lib;
      in rec {
        default = dockdns;

        dockdns = pkgs.buildGoModule {
          name = "dockdns";
          version = toString (self.shortRev or self.dirtyShortRev or self.lastModified or "unknown");
          buildInputs = nixpkgs.lib.lists.optionals pkgs.stdenv.isDarwin [pkgs.darwin.apple_sdk.frameworks.AppKit];
          src = lib.fileset.toSource {
            root = ./.;
            fileset = lib.fileset.unions [
              ./go.mod
              ./go.sum
              ./main.go
              ./internal
              ./templates
              ./static
            ];
          };
          vendorHash = "sha256-SBj77wu4wivrwc69E/9tSn4QV5DoqWkTjGc22r2Us/4=";
          meta.mainProgram = "dockdns";
        };

        dockdns-docker = pkgs.dockerTools.buildImage {
          name = "dockdns";
          config = {
            Cmd = ["${lib.getExe dockdns}"];
          };
        };
      });
  };
}
