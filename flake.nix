{
  description = "Devshell and Building ytdtd";

  inputs = {
    nixpkgs.url      = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url  = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        overlays = [];
        pkgs = import nixpkgs {
          inherit system overlays;
        };

        module = pkgs.buildGoModule {
          pname = "ytdtd";
          version = self.shortRev or "dirty";
          src = ./.;

          vendorHash = "sha256-Qd42J10roX69bGCnVYIoK3ioVJxZDZvn5Q/olZVP7QM=";

          nativeBuildInputs = [ pkgs.makeWrapper ];

          postFixup = ''
            wrapProgram $out/bin/ytdtd --prefix PATH : ${pkgs.lib.makeBinPath [ pkgs.yt-dlp pkgs.ffmpeg pkgs.imagemagick ]}
          '';
        };
      in
      {
        packages.default = module;
        packages.ytdtd = module;

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls

            yt-dlp
            imagemagick
            ffmpeg
          ];
        };
      }
    );
}
