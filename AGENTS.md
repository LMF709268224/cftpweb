# Agent Working Agreement

## Windows and Encoding

- The primary development environment is Windows. All commands and edits must be compatible with Windows paths and PowerShell.
- Treat repository text files as UTF-8. Do not use tools or commands that rewrite files with legacy Windows code pages such as GBK, ANSI, or the default Windows PowerShell 5.1 encoding.
- Do not use PowerShell (`run_command`, `Get-Content`, `Select-String`) to read or search files. You MUST ALWAYS use the native `view_file` and `grep_search` tools to prevent character encoding issues and avoid triggering terminal security prompts.
- Manual file edits must use the patch tool, not shell redirection, `Set-Content`, `Out-File`, or ad-hoc scripts that can silently change encoding.
- Never introduce or preserve Unicode replacement characters (`U+FFFD`) in source files. If a touched line already contains mojibake or `U+FFFD`, stop and repair that line as part of the change.
- Be careful with existing Chinese comments and strings. Do not rewrite nearby text unless required, and verify any touched Chinese text remains valid UTF-8.

## Delivery Expectations

- When making code changes, remove obsolete routes, handlers, and frontend calls that the change makes unused.
- Run the relevant backend and frontend checks before finishing whenever feasible.
- Create a git commit for completed coding work unless the user explicitly asks not to.

## Interface Contract Discipline

- Do not add guessed fallback or auto-fill behavior just to make an API call pass. If an interface rejects a request, first determine whether our request shape is wrong, the contract changed, or the microservice has a bug.
- Do not silently copy data between semantically different fields, especially persisted business JSON fields such as prices, IDs, status, or eligibility flags, unless the microservice contract explicitly requires that behavior.
- If a contract mismatch is unclear, document the exact request, response, and observed state before changing behavior. Prefer asking the microservice owner with concrete examples over mutating stored data from the frontend or BFF.
- Any compatibility fallback must be explicit, narrowly scoped, reviewed against the service contract, and called out in the final response with the reason it is safe.
