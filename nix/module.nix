self: { config, lib, pkgs, ... }:
let cfg = config.services.mochil;
in {
  options.services.mochil = {
    enable = lib.mkEnableOption "Mochil filesystem-first OPDS server";
    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.system}.default;
      description = "Mochil package to run.";
    };
    library = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "/srv/books";
      description = "Legacy single-library path. Use configFile for multiple libraries.";
    };
    configFile = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "/etc/mochil/config.json";
      description = "JSON configuration file for multiple filesystem or Calibre libraries.";
    };
    listen = lib.mkOption { type = lib.types.str; default = "0.0.0.0:8080"; };
    publicURL = lib.mkOption { type = lib.types.str; default = ""; example = "https://books.example.com"; };
    scanEvery = lib.mkOption { type = lib.types.str; default = "10m"; };
    environmentFile = lib.mkOption {
      type = lib.types.nullOr lib.types.str;
      default = null;
      example = "/run/secrets/mochil";
      description = "Optional file containing MOCHIL_USERNAME and MOCHIL_PASSWORD.";
    };
  };

  config = lib.mkIf cfg.enable {
    assertions = [{
      assertion = (cfg.configFile != null) != (cfg.library != null);
      message = "services.mochil requires exactly one of configFile or library";
    }];
    users.users.mochil = { isSystemUser = true; group = "mochil"; description = "Mochil service user"; };
    users.groups.mochil = {};
    systemd.services.mochil = {
      description = "Mochil ebook catalogue";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];
      serviceConfig = {
        User = "mochil";
        Group = "mochil";
        StateDirectory = "mochil";
        WorkingDirectory = "/var/lib/mochil";
        ExecStart = if cfg.configFile != null then
          "${cfg.package}/bin/mochil -config ${lib.escapeShellArg (toString cfg.configFile)}"
        else lib.concatStringsSep " " [
          "${cfg.package}/bin/mochil"
          "-library" (lib.escapeShellArg cfg.library)
          "-cache /var/lib/mochil/catalog.json"
          "-listen" (lib.escapeShellArg cfg.listen)
          "-scan-every" (lib.escapeShellArg cfg.scanEvery)
          "-public-url" (lib.escapeShellArg cfg.publicURL)
        ];
        EnvironmentFile = lib.optional (cfg.environmentFile != null) cfg.environmentFile;
        # The app's own /api/restart sends itself a clean exit when asked to restart (self-exec
        # on Linux now, see cmd/mochil/process_signal_unix.go — but this stays as a safety net for
        # whatever that call falls back to). "on-failure" treats a clean exit as success and never
        # restarts the unit, which is how a requested restart went dark forever before.
        Restart = "always";
        RestartSec = 2;
        NoNewPrivileges = true;
        PrivateTmp = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        ReadOnlyPaths = lib.optional (cfg.library != null) cfg.library;
      };
    };
  };
}
