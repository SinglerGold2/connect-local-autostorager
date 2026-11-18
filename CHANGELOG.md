# Changelog

## 0.1.0 - 2026-11-18

### Added

- A folder watcher: up to four supervised directories, each added with
  **[+] New supervised directory** and each with its own `successful/` and
  `failed/`.
- An import that goes through the storage API, so a file appears with its
  preview and slicer details exactly as a manual upload would. A name already
  taken is never overwritten; the module appends `-1`, `-2` and so on.
- A failure file beside every file that could not be imported, naming the step,
  the reason, the time and the path.
- A settings page for the folders, the check interval, the storage, the size
  limit and whether the source is moved after a commit.
- A manual page, and a dashboard card listing the last five imports.

### Notes

- The module asks for `read_storage` and `write_storage`, and for a database of
  its own, which keeps the settings across a restart.
- A folder on the host of a containerized server has to be mounted into the
  container first; the setting is the path inside the container.
