{
  description = "Crane Autoscaler for Kubernetes";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    devenv.url = "github:cachix/devenv";
  };

  outputs = inputs @ { flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devenv.flakeModule
      ];

      systems = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" ];

      perSystem =
        { config
        , self'
        , inputs'
        , pkgs
        , system
        , ...
        }: rec {
          devenv.shells = {
            default = {
              languages = {
                go.enable = true;
                go.enableHardeningWorkaround = true;
                go.package = pkgs.go_1_23;
              };

              pre-commit.hooks = {
                nixpkgs-fmt.enable = true;
                yamllint.enable = true;
                hadolint.enable = true;
              };

              packages = with pkgs;
                [
                  gnumake

                  operator-sdk
                  kubernetes-controller-tools
                  kubernetes-code-generator

                  kind
                  kubectl
                  kubectl-images
                  kustomize
                  kubernetes-helm
                  helm-docs

                  golangci-lint
                  yamllint
                  yamlfmt
                  hadolint
                  setup-envtest
                ]
                ++ [
                  self'.packages.opm
                ];

              scripts = {
                versions.exec = ''
                  go version
                  golangci-lint version
                  echo controller-gen $(controller-gen --version)
                  echo operator-sdk $(operator-sdk version)
                  echo opm $(opm version)
                  kind version
                  kubectl version --client
                  echo kustomize $(kustomize version --short)
                  echo helm $(helm version --short)
                '';
              };

              enterShell = ''
                versions
              '';

              # https://github.com/cachix/devenv/issues/528#issuecomment-1556108767
              containers = pkgs.lib.mkForce { };
            };

            ci = devenv.shells.default;
          };

          packages = {
            opm = pkgs.buildGoModule rec {
              pname = "opm";
              version = "1.49.0";

              src = pkgs.fetchFromGitHub {
                owner = "operator-framework";
                repo = "operator-registry";
                rev = "v${version}";
                sha256 = "sha256-47y9IOb8CJjHXb63k4L7Lxe8wHVilQT4ydv9xSJrGGs=";
              };

              vendorHash = "sha256-PBHBZuJdZ+6L1L/YHmrr5wYk90QE7jS91N7vmPlU0no=";

              subPackages = [ "cmd/opm" ];

              ldflags = [
                "-w"
                "-s"
                "-X github.com/${src.owner}/${src.repo}/cmd/opm/version.gitCommit==${src.rev}"
                "-X github.com/${src.owner}/${src.repo}/cmd/opm/version.opmVersion=${version}"
              ];
            };
          };
        };
    };
}
