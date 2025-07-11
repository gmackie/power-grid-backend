{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
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

    # Database tools (for local development)
    postgresql
    redis

    # YAML/JSON tools
    yq-go
    yamllint

    # Monitoring tools
    prometheus
    grafana
  ];

  shellHook = ''
    echo "🎯 Power Grid Game Server Development Environment"
    echo ""
    echo "Available tools:"
    echo "  • go $(go version | cut -d' ' -f3)"
    echo "  • docker $(docker --version | cut -d' ' -f3 | sed 's/,//')"
    echo "  • kubectl $(kubectl version --client --short 2>/dev/null | cut -d' ' -f3)"
    echo ""
    echo "Quick commands:"
    echo "  • make dev          - Start development server"
    echo "  • make test         - Run tests"
    echo "  • make docker-build - Build Docker image"
    echo "  • make docker-run   - Run in Docker"
    echo ""

    # Set up Go environment
    export GOPATH=$PWD/.go
    export PATH=$GOPATH/bin:$PATH
    mkdir -p $GOPATH

    # Set up development database URLs
    export DATABASE_URL="postgres://powergrid:powergrid@localhost:5432/powergrid?sslmode=disable"
    export REDIS_URL="redis://localhost:6379"

    # Set development environment
    export ENVIRONMENT="development"
    export PORT="4080"
  '';
}