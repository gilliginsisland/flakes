# git-remote-dropbox (Go)

A Go port of [Anish Athalye's git-remote-dropbox](https://github.com/anishathalye/git-remote-dropbox).
Use a Dropbox folder or shared folder as a Git remote, without the Dropbox desktop client.
Requires Go 1.26+ to build, and Git at runtime.

From the flakes repository root:

```sh
nix build .#git-remote-dropbox
```

```sh
go build -o bin/git-remote-dropbox .
# Put this checkout's bin directory on PATH, or install with:
go install .

git clone dropbox:///path/to/repo
# Or add a remote to an existing repository:
git remote add origin dropbox:///path/to/repo
git push -u origin main
```

There is one executable, `git-remote-dropbox`, using Go's standard `flag` package.
Git invokes it automatically for `dropbox://` URLs. Additional flags are:

```sh
git-remote-dropbox --version
git remote-dropbox --set-head main origin
```

## Authentication

Authentication happens when a remote operation needs it. The helper checks Git's
credential store without prompting for a password. If no credential is stored,
it prints a Dropbox OAuth authorization URL. Open that URL, authorize your
account, and paste the authorization code at Git's password prompt. That prompt
expects the one-time authorization code, not your Dropbox password or an access
token. Authentication input uses Git's askpass/terminal handling and never reads
the remote-helper protocol stream.

OAuth uses PKCE with the public client ID from Dropbox's JavaScript SDK demo,
`42zjexze6mfpf7x`; no client secret is used. The demo application determines the
available Dropbox permissions. After Dropbox accepts the authorization, the
refresh token is stored through `git credential approve`. The SDK automatically
refreshes access tokens. Revoked refresh tokens trigger a new OAuth flow;
network and server failures leave stored credentials intact.

Configure a credential helper if you want tokens retained between operations.
For example, macOS Keychain can be configured for Dropbox:

```sh
git config --global credential.https://dropbox.com.helper osxkeychain
```

Without a storage helper, authentication still works but must be repeated.
Storage security and retention are determined by the configured helper. There
is no application credential file, account registry, migration, or login manager.

Account aliases are part of the remote URL:

```sh
git clone dropbox://work@/projects/repo
```

Each alias uses a separate credential. It is a local label, not a Dropbox email
address. A URL without an alias uses the literal username `default`, so
`dropbox:///projects/repo` and `dropbox://default@/projects/repo` target the same
credential. Credentials use `protocol=https`, `host=dropbox.com`, the alias (or
`default`) as `username`, and the refresh token as `password`. The credential path
is omitted. Git handles `GIT_ASKPASS`, `core.askPass`, `SSH_ASKPASS`, and terminal
prompting; remote-helper stdin and stdout remain reserved for Git's protocol.

Inline access-token URLs are also supported and bypass credential storage.

## Git executable selection

Git supplies `GIT_EXEC_PATH` when invoking the helper. All Git subprocesses,
including credential operations, use `git` (`git.exe` on Windows) from that
directory. Missing, empty, or relative `GIT_EXEC_PATH` is an explicit error; there is no
PATH fallback or parent-process discovery. Run the default-branch flag through
`git remote-dropbox` as shown above so Git supplies the environment variable.

## Debugging

Enable diagnostic messages when running Git:

```sh
GIT_REMOTE_DROPBOX_DEBUG=1 git push dropbox
```

The trace identifies the running helper and Git executables, authentication
stages, Git commands, storage operations, upload chunks, HTTP status codes and
request IDs, and protocol operations. It does not log credentials, HTTP bodies,
file contents, or raw remote URLs.
Progress and diagnostics use stderr and respect redirection. Git passes its
`--quiet`/`--verbose` settings through the helper's `verbosity` protocol option,
and its terminal detection and `--progress`/`--no-progress` settings through the
separate `progress` option. Progress updates the same line after each staged
object, starting at zero, and ends with a newline when complete. Explicit
`--progress` enables progress even with redirected stderr or `--quiet`.
Stdout remains reserved for the remote-helper protocol. Push failures are
reported even when debugging is disabled. If an error has an empty
message, its type and structured details are shown instead.

## Compatibility and design

The remote layout matches the Python helper: loose objects under `objects/`,
refs under `refs/`, and a symbolic `HEAD`. Native Git plumbing handles local
objects and history. A single `git cat-file --batch` stream feeds a bounded queue
of 20 upload workers. Object data uploads into sessions in parallel; batches of
up to 1,000 sessions are committed serially with Dropbox's batch API. Large
objects use 50 MiB chunks, including recovery from incorrect offsets. Progress
shows staged objects as they upload and committed objects after each batch.
Reads use up to 20 concurrent transfers. The SDK
retries HTTP 429 rate limits until success or cancellation, honoring `Retry-After`
(or the response body delay), with a five-second fallback. Other transient API
failures retain the three-retry limit. Ctrl-C cancels requests and retry waits,
and interrupts protocol reads in an idle session.

All required objects are uploaded before a ref is published. Normal updates
check ancestry and use Dropbox revisions for compare-and-swap; creation uses
strict add, and force pushes overwrite. These guarantees apply per ref, not to
an entire multi-ref push. Deleting the default branch is refused. A concurrent
deletion during `--set-head` can require a manual retry, as in upstream.

SHA-1 repositories are supported; shallow clones are unsupported. Remote objects
remain loose, without packing or garbage collection. Do not modify the remote
through the Dropbox desktop client. Share the remote's root folder through
Dropbox to collaborate.

## License

MIT; see [LICENSE.md](LICENSE.md). Derived from Anish Athalye's MIT-licensed
implementation.
