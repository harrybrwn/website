{
  description = "hrry.me homelab monorepo";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
    gomod2nix = {
      url = "github:tweag/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.utils.follows = "utils";
    };
    rust-overlay = {
      url = "github:oxalica/rust-overlay";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "utils";
    };
    # For building rust packages
    naersk = {
      url = "github:nix-community/naersk";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    flake-compat = {
      url = "github:edolstra/flake-compat";
      flake = false;
    };
  };

  outputs = { self, nixpkgs, utils, gomod2nix, rust-overlay, naersk, flake-compat, ... }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [
        "x86_64-linux"
        "x86_64-darwin"
        "aarch64-linux"
        "aarch64-darwin"
        "armv7l-linux"
      ];
      nixpkgsFor = forAllSystems (system:
        import nixpkgs {
          inherit system;
          overlays = [
            gomod2nix.overlays.default
            rust-overlay.overlays.default
          ];
        });
      getRust = (pkgs:
        pkgs.rust-bin.stable.latest.default.override {
          targets = [ "x86_64-unknown-linux-gnu" ];
        });
    in
    {

      packages = forAllSystems (system:
        let
          version = "0.0.1";
          pkgs = nixpkgsFor.${system};
          naersk' = pkgs.callPackage naersk { };
          mockgen = import ./config/nix/mockgen.nix { inherit pkgs; };
          buildGo = name: overrides: pkgs.buildGoModule
            ({
              inherit version;
              name = name;
              subPackages = [ "./cmd/${name}" ];
              src = ./.;
              vendorSha256 = "sha256-y0WPBo8onZ9bshRRI8rSdDcqfvLG6B2srLek6gTAT2Q=";
              preBuild = "go generate ./...";
              doCheck = false;
              nativeBuildInputs = [ mockgen ];
            } // overrides);
        in
        {
          default = pkgs.buildEnv {
            inherit version;
            name = "hrry.me";
            paths = with self.packages.${system}; [ tools freeradius api geoip ];
          };
          tools = pkgs.buildEnv {
            inherit version;
            name = "tools";
            paths = with self.packages.${system}; [ mockgen kubeval mc provision ];
          };
          geoip = naersk'.buildPackage rec {
            inherit version;
            name = "geoip";
            src = ./.;
            cargoBuildOptions = o: o ++ [ "--package" name ];
          };
          api = buildGo "api" { };
          provision = buildGo "provision" { };
          inherit mockgen;
          kubeval = pkgs.callPackage ./config/nix/kubeval.nix { };
          mc = pkgs.callPackage ./config/nix/mc.nix { buildGoModule = pkgs.buildGo119Module; };
          freeradius = pkgs.callPackage ./config/nix/freeradius.nix {
            withPython3 = false;
            withMysql = true;
          };
        });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
          rust = getRust pkgs;
          go = pkgs.go_1_20;
        in
        {

          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              # Languages
              go
              rust
              nodejs
              # Infrastructure
              kubectl
              terraform
              kubernetes-helm # helm
              kube3d # k3d
              k9s
              ansible
              ansible-lint
              postgresql
              # Dev tools
              bmake
              rust-analyzer
              golangci-lint
              kube-linter
              operator-sdk # sdk for k8s operators
              # Shell utilities
              git
              jq
              curl
              shellcheck
              ripgrep
              tokei
              gomod2nix.packages.${system}.default # gomod2nix command line utility
              self.packages.${system}.tools
              # Python deps
              (python3.withPackages (ps: with ps; [
                requests
                paramiko
                boto3
                # poetry
              ]))
            ];

            shellHook = ''
              scripts/configure.sh
              alias bake=bin/bake k8s=bin/k8s
              export GOROOT="${go.outPath}/share/go"
              #PS1="$(echo $PS1 | sed -Ee 's/\\\$$/\(nix dev\) \\$/') "
            '';
          };
        });

      # The default package for 'nix build'. This makes sense if the
      # flake provides only one package or there is a clear "main"
      # package.
      defaultPackage = forAllSystems (system: self.packages.${system}.default);
    };
}
