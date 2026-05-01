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
    # Reject AI attribution lines in commit messages.
    no-ai-attribution = {
      enable = true;
      name = "no AI attribution in commit messages";
      stages = [ "commit-msg" ];
      language = "system";
      entry = toString (
        pkgs.writeShellScript "no-ai-attribution" ''
          set -eu
          msg_file="$1"
          if ${pkgs.gnugrep}/bin/grep -qiE \
            -e 'co-authored-by:.*(claude|anthropic|copilot|chatgpt|openai|gemini|cursor)' \
            -e 'noreply@anthropic\.com' \
            -e '🤖[[:space:]]*generated with' \
            -e 'generated with \[?claude' \
            -e 'generated with \[?github copilot' \
            "$msg_file"; then
            echo "error: commit message contains AI attribution — remove it before committing." >&2
            exit 1
          fi
        ''
      );
    };
  };
}
