{
  description = "Packer Provider";

  # Not a NixOS-specific project, so track nixpkgs-unstable rather than a release channel.
  inputs.nixpkgs.url = "nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let

      # to work with older version of flakes
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";

      version = builtins.substring 0 8 lastModifiedDate;

      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;

      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

    in
    {

      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          terraform-provider-packer = pkgs.buildGoModule {
            pname = "terraform-provider-packer";
            inherit version;
            src = ./.;

            # Bump this whenever the Go dependencies change. Set to
            # pkgs.lib.fakeHash to have the build report the expected value.
            vendorHash = "sha256-IVDayhTKlnLuXlRzxYj8A+T37Als6yP0wkPpsYHu/e0=";
          };
        });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [ go gopls gotools go-tools goreleaser ];
          };
        });

      defaultPackage = forAllSystems (system: self.packages.${system}.terraform-provider-packer);
    };
}
