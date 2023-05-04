{ pkgs ? import <nixpkgs> { } }:
pkgs.buildGoModule rec {
  pname = "mockgen";
  version = "1.6.0";
  src = fetchGit {
    url = "https://github.com/golang/mock.git";
    rev = "aba2ff9a6844d5e3289e8472d3217d5b3090f083";
  };
  subPackages = [ "mockgen" ];
  vendorSha256 = "sha256-5gkrn+OxbNN8J1lbgbxM8jACtKA7t07sbfJ7gVJWpJM=";
  CGO_ENABLED = 0;
  ldflags = [
    "-w"
    "-s"
    "-X main.version=${version}"
    "-X main.commit=${src.rev}"
    "-X main.date=unknown"
  ];
  meta = with pkgs.lib; {
    description = "GoMock is a mocking framework for the Go programming language";
    homepage = "https://github.com/golang/mock";
    license = licenses.asl20;
  };
}
