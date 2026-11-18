# <img src="artwork/icon.svg" width="40" align="center" alt=""> Auto-Storager

A Prusa Connect Local extension that watches up to four folders and hands every
print file it finds to the server's storage. The file arrives there like a
manual upload: with its preview, its slicer details and its print time. Nothing
is moved or deleted before the copy is complete.

## Installation

The extension is a normal module. Install it the way your server takes modules.

### Linux, server in a container

```bash
docker cp autostorager-0.1.0.tar.gz prusa-connect-local:/tmp/
docker exec -w /app prusa-connect-local /app/prusa_connect_local \
  module add github.com/singlergold2/connect-local-autostorager@0.1.0 \
  -from /tmp/autostorager-0.1.0.tar.gz -approve -database sqlite
docker restart prusa-connect-local
```

The `-database sqlite` is required: the settings a person edits on the page live
in a database of the module's own. The module asks for `read_storage` and
`write_storage`, because it looks up free names and writes the files.

### Linux, server as a process

```bash
./prusa_connect_local module add github.com/singlergold2/connect-local-autostorager@0.1.0 \
  -from autostorager-0.1.0.tar.gz -approve -database sqlite
```

Then restart the server.

### Windows

```powershell
cd C:\prusa-connect
.\prusa_connect_windows_amd64.exe module add github.com/singlergold2/connect-local-autostorager@0.1.0 `
  -from C:\Users\you\Downloads\autostorager-0.1.0.tar.gz -approve -database sqlite
```

Then restart the server. The module lands under
`config\config.d\modules\autostorager\bin`.

### macOS

```bash
cd ~/prusa-connect
./prusa_connect_macos_arm64 module add github.com/singlergold2/connect-local-autostorager@0.1.0 \
  -from ~/Downloads/autostorager-0.1.0.tar.gz -approve -database sqlite
```

Then restart the server. If macOS refuses a downloaded file, clear the
quarantine flag once with `xattr -d com.apple.quarantine <file>`.

## Configuration

Everything is set on the extension's own page: **Auto-Storager** in the sidebar.

### Linux, server in a container

A container sees only what is mounted into it, so create the folder on the host
first, add it as a volume, and enter the path *inside* the container.

```yaml
# docker-compose.yml, in the server's service
services:
  prusa-connect:
    volumes:
      - /srv/autostorager:/incoming
```

```bash
mkdir -p /srv/autostorager
docker compose up -d
```

The setting is then `/incoming`, not `/srv/autostorager`. The module creates
`successful/` and `failed/` inside it; on the host they appear as
`/srv/autostorager/successful` and `/srv/autostorager/failed`. Up to four mounts
may be added, one per supervised folder.

### Windows

The server is a normal process and reads the folder directly. Enter the full
path, such as `C:\Autostorager`, and save. `successful\` and `failed\` are created
inside it.

### macOS

Same as Windows, with a POSIX path such as `/Users/you/Autostorager`. No mount
is involved when the server runs as a process.

### The settings

| Setting | Meaning |
| --- | --- |
| Supervised folders | Up to four absolute paths, added with **+ New supervised directory**. Each has its own `successful/` and `failed/`. No folder means nothing is watched. |
| Check interval | Seconds between two looks. 60 s = 1 min. A short interval means constant requests, which costs I/O on a network share; five seconds is enough for most folders. |
| Storage | The storage instance files are imported into, `local` on most servers. |
| Largest file | In megabytes, 250 by default. A larger file goes to `failed/` with the reason. |
| Move on success | Move the source into `successful/` after the commit. |

The page stores these in the module's own database. The same values also appear
on the module's dashboard page; what is saved on the page wins and survives a
restart.

## What happens to a file

1. Each supervised folder is read every few seconds. Only `.gcode` and `.bgcode`
   files are taken.
2. A file is read once its size has held still across two checks, so a file a
   slicer is still writing is left alone.
3. The module asks the storage for a free name and appends `-1`, `-2` and so on
   if the name is taken. Nothing is ever overwritten.
4. The bytes are copied into the storage through the same API an upload uses.
   Only after that copy is committed is the source moved into `successful/`.
5. A file that cannot be imported is moved into `failed/`, and a text file
   beside it names the step, the reason, the time and the path.

## Checking it works

Drop a small `.gcode` into a supervised folder. Within about one interval it
should leave the folder, appear under `successful/`, and show up under **Storage**
with its preview. The **Recent imports** list on the page and the dashboard card
tell the same story, including a failure with its reason.

## Development

```bash
sh install.sh              # build the bundle and the binary, install into the demo
TARGET=app sh install.sh   # install into the real app instead
```

The build needs the component library of a Prusa Connect checkout; `install.sh`
builds it if `packages/svelte/dist` is missing.

## Licence

MIT.
