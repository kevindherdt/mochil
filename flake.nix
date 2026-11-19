{
  description = "A fast, filesystem-first OPDS 1.2 server";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in {
      packages = forAllSystems (system:
        let pkgs = import nixpkgs { inherit system; };
        in {
          default = pkgs.buildGoModule {
            pname = "opdshelf";
            version = "0.1.0";
            src = self;
            vendorHash = null;
            subPackages = [ "cmd/opdshelf" ];
            ldflags = [ "-s" "-w" ];
          };
        });
      apps = forAllSystems (system: {
        default = { type = "app"; program = "${self.packages.${system}.default}/bin/opdshelf"; };
      });
      nixosModules.default = import ./nix/module.nix self;
    };
}
