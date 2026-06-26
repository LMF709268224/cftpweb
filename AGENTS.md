# Agent Working Agreement

## Windows and Encoding

- The primary development environment is Windows. All commands and edits must be compatible with Windows paths and PowerShell.
- Treat repository text files as UTF-8. Do not use tools or commands that rewrite files with legacy Windows code pages such as GBK, ANSI, or the default Windows PowerShell 5.1 encoding.
- Before reading or displaying files that contain non-ASCII text in PowerShell, set UTF-8 output explicitly when needed:

```powershell
$OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
```

- When using PowerShell to inspect UTF-8 files, prefer `Get-Content -Encoding UTF8`.
- Manual file edits must use the patch tool, not shell redirection, `Set-Content`, `Out-File`, or ad-hoc scripts that can silently change encoding.
- Never introduce or preserve Unicode replacement characters (`U+FFFD`) in source files. If a touched line already contains mojibake or `U+FFFD`, stop and repair that line as part of the change.
- Be careful with existing Chinese comments and strings. Do not rewrite nearby text unless required, and verify any touched Chinese text remains valid UTF-8.

## Delivery Expectations

- When making code changes, remove obsolete routes, handlers, and frontend calls that the change makes unused.
- Run the relevant backend and frontend checks before finishing whenever feasible.
- Create a git commit for completed coding work unless the user explicitly asks not to.
