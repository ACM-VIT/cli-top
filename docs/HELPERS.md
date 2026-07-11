# Helper Architecture

The `helpers` package contains shared infrastructure used by the CLI commands and
feature implementations. Keep feature-specific parsing and business rules in
`features`; add code here only when multiple features genuinely share it.

## HTTP and VTOP sessions

- `GetHTTPClient` returns the shared, TLS-verifying HTTP client.
- `FetchReq` handles normal VTOP requests, status validation, request timeouts,
  and one bounded relogin retry with refreshed cookies and CSRF data.
- `FetchReqClient` and `FetchReqClientWithContext` support callers that need a
  specific client or cancellation context, such as concurrent downloads.
- `ValidateVtopURL` rejects requests that do not target VTOP.
- `SetVtopHeaders` applies the common browser-compatible request headers.

Do not create private HTTP clients inside features. Using the shared transport
keeps timeout, TLS, proxy, connection reuse, and test behavior consistent.

## Configuration and semesters

- `ConfigFilePath` resolves `cli-top-config.env` from the working directory,
  executable directory, or the user's config directory.
- Semester results are cached per registration number and persisted in the
  config file.
- `GetSemDetails` tries the attendance page first and the course-page endpoint
  as a fallback.
- `SelectSemester` renders the shared selector and supports deterministic
  numeric selection for command flags and proxy execution.

## Terminal output and selection

- `CommandRunner` and `CommandRunnerE` provide the animated shimmer headline,
  elapsed time, and success/failure status around commands.
- `Print`, `Println`, and `Printf` safely stop the active headline before other
  output is written.
- `PrintTable` formats terminal tables without mutating caller data and can emit
  structured snapshots for proxy responses.
- `TableSelector` and `TableSelectorFuzzy` provide numeric and fuzzy selection.
- `CLI_TOP_PROXY_MODE=1` disables interactive UI and OS file/folder opening while
  preserving structured table and selection output.

## Update controls

- `CheckKillSwitch`, `CheckUpdate`, and `CheckUpdateSilently` share one validated,
  timeout-bound `latest.json` fetch path.
- Successful kill-switch responses are cached briefly to avoid duplicate startup,
  facility, and CAPTCHA requests.
- `CLI_TOP_LATEST_JSON_URL` is an integration-test override.
- Kill-switch values currently mean:
  - `0`: normal operation
  - `1`: manual CAPTCHA entry
  - `2`: decommissioned client
  - `3`: open VTOP in the browser
  - `4`: facility registration is view-only

## Files and calendars

- `GetOrCreateDownloadDir` keeps generated files under `CLI-TOP Downloads`.
- `SanitizeFilename` normalizes downloaded file names.
- `GetFileExtension` uses response metadata and one content inspection path,
  including OOXML ZIP detection.
- `GenerateICSFileDateOnly` and `VenueAdd` write calendar files.
- `UploadICSFile` uploads generated calendars when a feature offers subscription
  links.

## CAPTCHA solver

`SolveCaptcha` accepts the VTOP JPEG data URL. Automated solving runs entirely in
memory. Manual mode writes `captcha.jpg` with user-only permissions and prompts
once. Invalid data and unexpected image dimensions fail without panicking.

## Testing expectations

Tests replace the shared HTTP transport with strict path-aware round trippers and
must restore it after use. Tests must not open browsers, submit real VTOP forms,
write into the user's home directory, or depend on a globally installed binary.
