{
  description = "SIGE - Sistema de Isolamento e Gerenciamento de Execução";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "sige";
          version = "0.1.0";
          src = ./.;

          vendorHash = null;

          nativeBuildInputs = with pkgs; [
            pkg-config
            gcc
          ];

          buildInputs = with pkgs; [
            libseccomp
          ];

          env = {
            CGO_ENABLED = "1";
          };

          meta = with pkgs.lib; {
            description = "Sandbox de isolamento seguro para execução de código não confiável";
            license = licenses.mit;
            platforms = platforms.linux;
          };
        };

        packages.fhs = pkgs.buildFHSEnv {
          name = "sige-fhs";
          targetPkgs = pkgs: with pkgs; [
            go
            gcc
            pkg-config
            gopls
            gotools
            delve
            golangci-lint
            docker-compose
            python3
            bash
            coreutils
            libseccomp
          ];
          runScript = "bash";
        };

        devShells.default = pkgs.mkShell {
          name = "sige-dev";

          nativeBuildInputs = with pkgs; [
            go
            gcc
            pkg-config

            gopls
            gotools
            delve
            golangci-lint

            docker-compose

            python3
          ];

          buildInputs = with pkgs; [
            libseccomp
          ];

          CGO_ENABLED = "1";

          shellHook = ''
            echo ""
            echo "🔒 SIGE - Ambiente de desenvolvimento NixOS"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo "  Go:      $(go version | awk '{print $3}')"
            echo "  Python:  $(python3 --version 2>/dev/null || echo 'não encontrado')"
            echo "  CGO:     habilitado (libseccomp)"
            echo "  Módulo:  $(go list -m 2>/dev/null || echo 'sige')"
            echo ""
            echo "  Comandos úteis:"
            echo "    go build -o sige main.go   → Compilar"
            echo "    ./sige init-config         → Gerar config.json"
            echo "    ./sige run-task --help      → Ver opções"
            echo "    go test ./...              → Rodar testes"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo ""
          '';
        };

        devShells.fhs = self.outputs.packages.${system}.fhs.env;
      }
    );
}
