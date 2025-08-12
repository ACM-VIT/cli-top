# RELEASE.md

## Steps After Your Pull Request to Upstream `dev` Is Merged

Once your pull request from your fork's `dev` branch to the upstream `dev` branch is accepted (including the version number change), follow these steps to complete the release:

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