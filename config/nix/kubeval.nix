{ lib, buildGoModule, fetchFromGitHub }:
let
  owner = "instrumenta";
in
buildGoModule rec {
  pname = "kubeval";
  version = "0.16.1";
  src = fetchGit {
    url = "https://github.com/${owner}/kubeval.git";
    ref = "refs/tags/v${version}";
    rev = "f5dba6b486fa18b9179b91e15eb6f2b0f7a5a69e";
  };
  doCheck = false;
  vendorSha256 = "sha256-OAFxEb7IWhyRBEi8vgmekDSL/YpmD4EmUfildRaPR24=";
  CGO_ENABLED = 0;
  ldflags = [
    "-w"
    "-s"
    "-X main.version=${version}"
    "-X main.commit=${src.rev}"
    "-X main.date=unknown"
  ];
  meta = {
    description = "Validate your Kubernetes configuration files, supports multiple Kubernetes versions";
    homepage = "https://kubeval.com/";
    changelog = "https://github.com/${owner}/kubeval/releases/tag/v${version}";
    license = lib.licenses.asl20;
  };
}
