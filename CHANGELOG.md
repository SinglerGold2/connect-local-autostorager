# Changelog

## 0.2.2 - 2026-11-20

The first public release.

### Watches folders

- Up to four supervised directories, each added with **+ New supervised
  directory**.
- Each folder has its own `successful/` and `failed/`.
- The folder is read every few seconds; a file is taken only after its size has
  held still across two checks, so a slicer still writing it is left alone.
- Only regular `.gcode` and `.bgcode` files at the top level are taken.
  Subfolders are not searched. Symbolic links, directories, devices, pipes and
  sockets are skipped; a link stays where it is and is never followed.
- A larger file than the size limit goes to `failed/` with the reason.

### Imports

- The file goes through the storage API, so it appears with its preview and
  slicer details exactly as a manual upload would.
- A name already taken is never overwritten; the module appends `-1`, `-2` and
  so on.
- Only after the copy is committed is the source moved into `successful/`.
- A file that cannot be imported goes to `failed/`, with a text file beside it
  naming the step, the reason, the time and the path.
- The source is moved only if **Move on success** is on.

### Dashboard card

- A card lists the last imports with a green dot for a committed import and a
  red dot for a failure, each with an accessible label.
- The card says "Autostorager" when no title is set.

### Three languages

- English, German and Czech for the page and the manual.
- `language` on the extension settings accepts `system`, `en`, `de` or `cs`.
  `system` follows the browser.
- Changing the setting restarts the module; the manual's EN, DE and CZ buttons
  switch the language for the visit.

### Users and rights

- The manual and the README explain the PCL accounts and the `viewer`,
  `operator` and `admin` roles, the grant on `org:local`, and the difference
  between OS admin and PCL admin on Linux, macOS and Windows.
- The manual opens with this block as IMPORTANT.
- The page's Save takes `operate_resource` on `resource_kind:sink`; a
  read-only viewer can see the page but cannot change it.

### The manual

- Settings, watched folders, interval, storage, size limit, moving the source,
  the language choice, the rights and the file rules, in the manual.

### Security

- The watcher takes only regular files. A symbolic link is skipped at the scan,
  opened without following it where the platform allows, and checked again on
  the opened descriptor before it is read.
- A non-regular source is never moved into `successful/` or `failed/`.
- Tests cover the scan, the open and the move with a symbolic link, and keep a
  regular `.gcode` import as the counter-check.

## Pre-release history

Earlier 0.1.0 and 0.2.x builds were internal pre-releases and were never
published. Their tags remain in the repository; 0.2.2 is the first public
release.
