# RELEASE.md

## Steps After Your Pull Request to Upstream `dev` Is Merged

Once your pull request from your fork's `dev` branch to the upstream `dev` branch is accepted (including the version number change), follow these steps to complete the release:

---

## Website Release Automation

The `GoBuilder.yml` workflow now pushes the packaged binaries directly to the public [`cli-top-website`](https://github.com/technical-director-acmvit/cli-top-website) repo and opens a pull request with the updated download links. To let the automation run end-to-end you must:

1. **Add a `WEBSITE_REPO_TOKEN` repository secret** that contains a Personal Access Token with `repo` scope. The PAT must have push/PR permissions on `technical-director-acmvit/cli-top-website`.
2. **Keep `release_metadata.json` up to date** on the private CLI repository before triggering `GoBuilder`.
   - `version` must match `debug/debug.go`.
   - `releaseDate` can be blank; it defaults to the current UTC date.
   - `killSwitch` lets you bump the remote kill-switch value (defaults to `4`).
   - `changes` is the bullet list shown on the website and in the CLI update notification.

When the workflow completes it will automatically:

- Download the multi-platform artifacts produced by the build jobs.
- Zip/rename them into `buildFiles/vX.Y.Z` inside `cli-top-website`.
- Update `latest.json` and prepend a release entry to `releases.json` using `release_metadata.json`.
- Push the changes to a branch named `auto/release-vX.Y.Z` and open (or update) a PR on the website repo. You only need to review and merge it.

---

### 1. Sync Your Local Repository

- Fetch the latest changes from upstream:
  ```sh
  git fetch upstream
  ```
- Then checkout the upstream `main` branch locally:
  ```sh
  git checkout -b upstream-main upstream/main
  ```
- If you want to update your local `main` with upstream:
  ```sh
  git checkout upstream-main
  git pull upstream main
  ```
- Then checkout the upstream `dev` branch locally:
  ```sh
  git checkout -b upstream-dev upstream/dev
  ```
- If you want to update your local `dev` with upstream:
  ```sh
  git checkout upstream-dev
  git pull upstream dev
  ```

---

### 1. Update Version in Source

- Commit the version update to the `upstream-dev` branch:
  ```sh
  git checkout upstream-dev
  git add debug/debug.go
  git commit -m "Bump version to vX.Y.Z"
  ```
---

### 1.1 Update `release_metadata.json`

- Edit the `release_metadata.json` file at the repository root:
  ```json
  {
    "version": "2.9.10",
    "releaseDate": "2025-11-17",
    "killSwitch": 4,
    "changes": [
      "Now you see this notification!",
      "Hotfix for auto-update",
      "Table responsiveness improvements"
    ]
  }
  ```
- Ensure the `version` matches `debug/debug.go`. Set the release notes bullets you want the website + CLI highlight to display. Commit this file alongside the version bump if anything changed.

---

### 2. Merge Upstream `dev` Into Upstream `main`

- Make sure you are on your local `main` branch:
  ```sh
  git checkout upstream-main
  ```
- Merge the latest changes from upstream `dev`:
  ```sh
  git merge upstream/dev
  ```
- Push the updated `main` branch to upstream:
  ```sh
  git push upstream main
  ```

---

### 3. Tag the Release

- Create a new tag that matches the version in `debug.go`:
  ```sh
  git tag vX.Y.Z
  git push upstream main --tags
  ```

---

### 4. Monitor the Release Workflow

- Go to GitHub Actions in the upstream repository.
- Confirm that the "Release with GoReleaser" workflow completes successfully.
- Check the release summary and artifacts.

---

### 5. Publish Release Notes (Optional)

- Edit the release on GitHub to add detailed release notes if needed.

---

**Note:**  
Do not skip any steps. The release workflow depends on the version in `debug.go
