use std::fs;

use zed::LanguageServerId;
use zed_extension_api as zed;

struct KlarExtension {
    cached_binary_path: Option<String>,
}

impl zed::Extension for KlarExtension {
    fn new() -> Self {
        Self {
            cached_binary_path: None,
        }
    }
    fn language_server_command(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command, String> {
        // `klar lsp`
        Ok(zed::Command {
            command: self.language_server_binary_path(language_server_id, worktree)?,
            args: vec!["lsp".to_string()],
            env: worktree.shell_env(),
        })
    }
}

impl KlarExtension {
    fn language_server_binary_path(
        &mut self,
        language_server_id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<String, String> {
        if let Some(path) = worktree.which("klar") {
            return Ok(path);
        }
        if let Some(path) = &self.cached_binary_path {
            if fs::metadata(path).map_or(false, |stat| stat.is_file()) {
                return Ok(path.clone());
            }
        }

        // TODO: Download Klar binary from GitHub
        _ = language_server_id;
        Err(
            "Klar isn't installed. Please install from https://github.com/ProCode-Software/klar"
                .to_string(),
        )
    }
}

zed::register_extension!(KlarExtension);
