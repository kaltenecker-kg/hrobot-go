{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
{
  # Provide GNU sed on macOS so scripts using GNU-only flags work consistently.
  scripts.sed.exec = ''
    ${pkgs.gnused}/bin/sed "$@"
  '';

  # https://devenv.sh/languages/
  languages.go.enable = true;

  git-hooks.excludes = [
    ".devenv"
    "vendor"
  ];

  # https://devenv.sh/reference/options/#git-hooks
  git-hooks.hooks = {
    # Go files
    golangci-lint.enable = true;
    # Nix files
    nixfmt-rfc-style.enable = true;
    # Github Actions
    actionlint.enable = true;
    # Markdown files
    markdownlint = {
      enable = true;
      settings.configuration = {
        # Max 130 line length, except if it's code
        MD013 = {
          line_length = 130;
          code_blocks = false;
        };
        # Allow bare URLs in documentation
        MD034 = false;
      };
    };
    # Try not to leak secrets
    ripsecrets.enable = true;
  };
}
