{ lib, buildGoModule }:
let
  releaseDate = "2023-04-12T02-21-51Z";
in
buildGoModule rec {
  pname = "mc";
  version = "RELEASE.${releaseDate}";
  src = fetchGit {
    url = "https://github.com/minio/mc.git";
    ref = "refs/tags/${version}";
    rev = "1843717c57fb87612469b7610344a7d49d97a497";
  };
  doCheck = false;
  vendorSha256 = "sha256-d8cC/exdM7OMGE24bN00BVE3jqE1tj6727JiON/aJkc=";
  CGO_ENABLED = 0;
  ldflags =
    let shortCommit = builtins.substring 0 12 src.rev;
    in
    [
      "-w"
      "-s"
      "-X github.com/minio/mc/cmd.Version=${releaseDate}"
      "-X github.com/minio/mc/cmd.CopyrightYear=2023"
      "-X github.com/minio/mc/cmd.ReleaseTag=${version}"
      "-X github.com/minio/mc/cmd.CommitID=${src.rev}"
      "-X github.com/minio/mc/cmd.ShortCommitID=${shortCommit}"
    ];
  meta = {
    description = "";
    homepage = "";
    changelog = "https://github.com/minio/mc/releases/tag/v${version}";
    license = lib.licenses.agpl3;
  };
}
