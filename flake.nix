{
  description = "A Nix-flake-based development environment";

  inputs.nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/0.1.0.tar.gz";

  outputs =
    { self, nixpkgs, }:
    let
      goVersion = 21; # Change this to update the whole stack
      overlays = [
        (final: prev: {
          go = prev."go_1_${toString goVersion}";
        })
      ];
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forEachSupportedSystem = f:
        nixpkgs.lib.genAttrs supportedSystems (system:
          f {
            pkgs = import nixpkgs { inherit overlays system; };
          });
    in
    {
      devShells = forEachSupportedSystem ({ pkgs, }: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            # go 1.21 (specified by overlay)
            go
            # goimports, godoc, etc.
            gotools
            # https://github.com/golangci/golangci-lint
            golangci-lint
            # go language server
            gopls
            gotestsum
            air
            sqlite
            just
          ];

          buildInputs = with pkgs; [
            sqlite-vss
          ];

          LOAD_VECTOR0 = ".load '${pkgs.sqlite-vss}/lib/vector0'";
          LOAD_VSS0 = ".load '${pkgs.sqlite-vss}/lib/vss0'";
        };
      });

      # So really, this flake is for devs to dogfood with, if
      # you're an end user you should be prepared for this flake to not
      # build periodically.
      packages = forEachSupportedSystem ({ pkgs, ... }: rec {
        pocketvector =
          let
            pname = "pocketvector";
            patchLibs =
              if pkgs.stdenv.isDarwin
              then ''
                install_name_tool -add_rpath "@loader_path/../lib" $out/bin/${pname}
              ''
              else ''
                ${pkgs.lib.getExe' pkgs.patchelf "patchelf"} --set-rpath "\$ORIGIN/../lib" $out/bin/${pname}
              '';
          in
          pkgs.buildGoModule {
            name = pname;

            src = ./.;
            CGO_ENABLED = 1;
            doCheck = false;

            vendorHash = "sha256-8cLOnMiOqhGWMh54IciASClsbLMJCrv8sKO0T6CxAAw="; # pkgs.lib.fakeHash;

            postInstall = ''
              ${patchLibs}
              mkdir $out/lib
              cp ${pkgs.sqlite-vss}/lib/vector0.* $out/lib
              cp ${pkgs.sqlite-vss}/lib/vss0.* $out/lib
            '';
          };
        default = pocketvector;
      });

      overlays.default = final: prev: {
        inherit (self.packages.${final.system}) pocketvector;
      };
    };
}
