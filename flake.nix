{
  description = "Power Grid Game Server";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        powergrid-server = pkgs.buildGoModule rec {
          pname = "powergrid-server";
          version = "0.1.0";

          src = ./.;

          vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="; # Update with actual hash

          buildInputs = with pkgs; [ ];

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
            "-X main.commit=${self.rev or "dirty"}"
            "-X main.date=${self.lastModifiedDate or "unknown"}"
          ];

          postInstall = ''
            mkdir -p $out/share/powergrid
            cp -r maps $out/share/powergrid/
            cp config.yml $out/share/powergrid/
          '';

          meta = with pkgs.lib; {
            description = "Power Grid board game server implementation";
            homepage = "https://github.com/your-username/power_grid_game";
            license = licenses.mit;
            maintainers = [ ];
          };
        };

      in
      {
        packages = {
          default = powergrid-server;
          powergrid-server = powergrid-server;
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go development
            go_1_21
            gopls
            go-tools
            golangci-lint
            delve

            # Build tools
            gnumake
            git

            # Container tools
            docker
            docker-compose
            kubectl
            helm

            # Development tools
            jq
            curl
            wget
            netcat

            # Testing tools
            k6
            websocat

            # Database tools
            postgresql
            redis

            # YAML/JSON tools
            yq-go
            yamllint

            # Monitoring tools
            prometheus
            grafana

            # Nix tools
            nixpkgs-fmt
            nil
          ];

          shellHook = ''
            echo "🎯 Power Grid Game Server Development Environment (Nix Flake)"
            echo ""
            echo "Available tools:"
            echo "  • go $(go version | cut -d' ' -f3)"
            echo "  • docker $(docker --version | cut -d' ' -f3 | sed 's/,//')"
            echo "  • kubectl $(kubectl version --client --short 2>/dev/null | cut -d' ' -f3 || echo "not configured")"
            echo ""
            echo "Nix commands:"
            echo "  • nix build         - Build the server package"
            echo "  • nix run           - Run the server"
            echo "  • nix develop       - Enter development shell (you're here!)"
            echo "  • nix flake check   - Check flake validity"
            echo ""
            echo "Development commands:"
            echo "  • make dev          - Start development server"
            echo "  • make test         - Run tests"
            echo "  • make docker-build - Build Docker image"
            echo ""

            # Set up Go environment
            export GOPATH=$PWD/.go
            export PATH=$GOPATH/bin:$PATH
            mkdir -p $GOPATH

            # Set up development environment
            export ENVIRONMENT="development"
            export PORT="4080"
            export DATABASE_URL="postgres://powergrid:powergrid@localhost:5432/powergrid?sslmode=disable"
            export REDIS_URL="redis://localhost:6379"
          '';
        };

        # NixOS module for the server
        nixosModules.powergrid-server = { config, lib, pkgs, ... }:
          with lib;
          let
            cfg = config.services.powergrid-server;
          in
          {
            options.services.powergrid-server = {
              enable = mkEnableOption "Power Grid Game Server";

              package = mkOption {
                type = types.package;
                default = powergrid-server;
                description = "The powergrid-server package to use";
              };

              port = mkOption {
                type = types.port;
                default = 4080;
                description = "Port to listen on";
              };

              host = mkOption {
                type = types.str;
                default = "127.0.0.1";
                description = "Host to bind to";
              };

              configFile = mkOption {
                type = types.path;
                default = "${cfg.package}/share/powergrid/config.yml";
                description = "Path to configuration file";
              };

              mapsDir = mkOption {
                type = types.path;
                default = "${cfg.package}/share/powergrid/maps";
                description = "Path to maps directory";
              };

              user = mkOption {
                type = types.str;
                default = "powergrid";
                description = "User to run the service as";
              };

              group = mkOption {
                type = types.str;
                default = "powergrid";
                description = "Group to run the service as";
              };
            };

            config = mkIf cfg.enable {
              users.users.${cfg.user} = {
                isSystemUser = true;
                group = cfg.group;
                description = "Power Grid Game Server user";
              };

              users.groups.${cfg.group} = {};

              systemd.services.powergrid-server = {
                description = "Power Grid Game Server";
                wantedBy = [ "multi-user.target" ];
                after = [ "network.target" ];

                serviceConfig = {
                  ExecStart = "${cfg.package}/bin/powergrid-server -config ${cfg.configFile}";
                  User = cfg.user;
                  Group = cfg.group;
                  Restart = "always";
                  RestartSec = "10s";

                  # Security settings
                  NoNewPrivileges = true;
                  PrivateTmp = true;
                  ProtectSystem = "strict";
                  ProtectHome = true;
                  ProtectKernelTunables = true;
                  ProtectControlGroups = true;
                  RestrictSUIDSGID = true;
                  RemoveIPC = true;
                  RestrictRealtime = true;
                };

                environment = {
                  HOST = cfg.host;
                  PORT = toString cfg.port;
                };
              };

              networking.firewall.allowedTCPPorts = [ cfg.port ];
            };
          };
      }
    );
}