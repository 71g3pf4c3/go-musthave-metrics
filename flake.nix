{
  description = "go-musthave-metrics dev shell";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = import nixpkgs { inherit system; };
      in {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_26
            gopls
            delve
            gotools
            golangci-lint
            gnumake
            git
            jq
            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
          ];

          shellHook = ''
            export GOPATH="$PWD/.gopath"
            export GOBIN="$GOPATH/bin"
            export GOMODCACHE="$GOPATH/pkg/mod"
            export GOCACHE="$PWD/.gocache"
            export PATH="$GOBIN:$PATH"

            mkdir -p "$GOBIN" "$GOMODCACHE" "$GOCACHE"

            if ! command -v mockgen >/dev/null 2>&1; then
              go install github.com/golang/mock/mockgen@v1.6.0
            fi

            echo "dev shell ready: go $(go version | awk '{print $3}')"
          '';
        };
      });
}
