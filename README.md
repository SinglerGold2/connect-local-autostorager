# Autostorager

A Prusa Connect module with both halves: a Go process serving one route,
and a browser bundle drawing on it.
`make new-module` wrote this skeleton.

Its namespace is `com.github.aux.singlergold2.connect-local-autostorager`, derived by the host from the repository
address `github.com/singlergold2/connect-local-autostorager`. You never write the namespace: every address the module
answers on is built from it.

| path | purpose |
| --- | --- |
| `module.yaml` | the manifest: the kind and the version, and the `ui` block with the ceiling this bundle may register into |
| `install.sh` | build, install and see it. The one command below |
| `CHANGELOG.md` | what an operator sees change when they upgrade this module |
| `.vscode/extensions.json` | the editor extensions for the languages in here |
| `main.go` | the process: the handshake, the lifecycle set, the settings and the route |
| `faults.go` | every code this module refuses with, each documented once |
| `main_test.go` | the route answering, the refusal carrying its code, and the manifest's default |
| `openapi.json` | the fragment describing that route, with every path relative to the mount |
| `ui/src/index.ts` | the bundle's entry: it receives the host object and registers what the manifest declares |
| `ui/src/page.svelte` | `autostorager`: a page of its own, with an entry in the sidebar |
| `ui/src/styles.css` | this module's own Tailwind. Four lines, and only one emits |
| `ui/src/locales/en.json` | every text this module draws. No component writes an English string |

## Installation

```bash
sh install.sh
```

That builds the bundle and this platform's binary, and installs the release with
`prusa-connect module add` into the demo's data, under `bin/autostorager-demo`
in the checkout. `TARGET=app sh install.sh` installs it into the real app's
data, `bin/autostorager-app`, instead. What the tools print goes to `install.log`
there, and a good run prints one line saying where the module is installed.

To see it, run this in the checkout. `-dir` is this folder, `../autostorager` when
it sits beside the checkout:

```bash
go run ./cmd/new-module -start demo -dir ../autostorager
```

It starts the demo on a free port with this module installed, and this
module's dev server in front of it. It prints the address to open and the
account to sign in with. `-start app` starts the real app instead, which asks
for an account on its first run. Ctrl+C stops everything it started.

Paste what the command printed on your machine under this paragraph. A README
that quotes real output is one a reader can check; one that describes the output
is one they have to trust.

Run `sh install.sh` again after a change to `module.yaml` or to the process.
`module add` records the bytes it installed and the host serves nothing it
cannot verify against that record, so a rebuilt release is a new install. A
core that is already running read its declarations when it started, so it is
restarted too.

## Iterating

`-start` runs the module's own dev server, `pnpm dev` in `ui/`. Open the
address it prints, not the core's. The page is the core's app and this module
is served from `src/`, so an edit to a component shows up without a reload.
Run by hand, `pnpm dev` reads the core's address from `PCL_CORE_URL`,
`http://localhost:8080` by default.

Install again after changing `module.yaml`: the core reads what a module
contributes when it is installed.

## Configuration

`options:` in `module.yaml` names every setting this module reads, each with its
type, its default and one sentence saying what it controls. Nothing here reads
an environment variable, so that list is the whole of what an operator can set.

The host applies the defaults, writes the instance's `config:` block over them
key by key, and sends the result on `plugin.configure` before the first request
arrives. A key the manifest does not name is refused before the process starts.

```yaml
resources:
  sink:
    autostorager:
      config:
        message: Ready.
```

## Permissions

This module declares no `permissions:`, because it acts only for the person the
host authorized and does nothing of its own: no poll, no sync loop, no
background work. The host takes `view_resource` on the instance before a read
and `operate_resource` before a write, and `openapi.json` states that on each
operation.

A module that does work with nobody behind it adds `permissions:` to
`module.yaml`, one line per permission saying what it lets the module do. An
operator reads that list before `module add -approve` writes it into the
declaration and grants the module what the list names. The install prints every
grant it wrote, and `module remove` takes them back.

## Errors

When the module cannot do what a request asks, it answers with an error. The
error has a `code` and a `title` in English. Every code of this module starts
with `com.github.aux.singlergold2.connect-local-autostorager`.

A program that calls the module should check the `code`. The title is for
people. You can change the wording of a title without breaking a caller.

`faults.go` lists every code, with what went wrong and what the caller can do
about it. Add a new code in that file. The page in the browser shows an error
through `describeFault`, which looks up the translation for the code and shows
the English title when it finds none.

## What the pieces are

The bundle's build writes `ui/dist`, and that directory is already the release
layout: `module.yaml` at its root and the entry at `ui/index.js`, with the
binary `autostorager_<GOOS>_<GOARCH>` beside them. `-from ui/dist` is what the install
reads. A `checksums.txt` is optional and verified when it is there; a release
you just built has nothing to prove against.

```bash
prusa-connect module add github.com/singlergold2/connect-local-autostorager@0.1.0 -from ui/dist
prusa-connect module list
prusa-connect module remove github.com/singlergold2/connect-local-autostorager
```

Without `-approve` the declaration is written switched off, and the command
prints what the module asked for so an operator can read it first.

## The ceiling

`contributes` in `module.yaml` is the whole of what this bundle may register.
A `host.register` call the block does not declare throws, and the host drops
this module whole. The generator wrote both sides from one answer:

- `pages: autostorager` ← `host.register.page('autostorager', Page)`

Add one to the manifest and the other to the entry, or neither.

## Styling

`ui/src/styles.css` carries four directives and only the third emits anything.
They let this module name any Tailwind utility the library's theme defines,
without shipping Tailwind's preflight into the host page. The page behind them
is `styling-a-module` in the core's documentation.

## Texts

Every text a person reads is a key in `ui/src/locales/en.json`, read with
`host.t('<key>')`. The host merges that file under this module's namespace, so
a key here may be one the core already names. Copy the file to `<tag>.json` to
translate it.

## Type the route in the browser

The bundle reads nothing from the process half yet. Generate a typed client
from the fragment and bind it onto the client the host handed over:

```bash
prusa-connect-generate-client-sdk -module \
  -document openapi.json -namespace com.github.aux.singlergold2.connect-local-autostorager \
  -name autostorager-client -out ./client
```

The bundle then imports `bind` from that package and calls
`bind(host.client).readAutostorager()`. There is no `fetch` and no token in a
module's browser half: the credential is the session the person already has.

## Troubleshooting

| what you see | what it means |
| --- | --- |
| `module add` refuses the release | `ui/dist` is not the release layout. `module.yaml` sits at its root and the entry at `ui/index.js` |
| the module is listed and switched off | `module add` ran without `-approve`. Read what it printed, then switch the instance on |
| the sidebar entry is missing | the app refused the bundle. Open the browser console: a registration `contributes` does not declare is refused there by name |
| a screen shows a key instead of a sentence | the key is missing from `ui/src/locales/en.json` |
| a response carries a code and an English sentence | the process refused the request. Look the code up in `faults.go` |
| a change does not show | the install was not run again. Every change is a new install |

## Mounting the folder on Linux

The server usually runs in a container, and a container sees only what is
mounted into it. A folder on the host is therefore invisible until it is handed
over, and the path this module watches is the path *inside* the container.

```yaml
# docker-compose.yml, in the server's service
services:
  prusa-connect:
    volumes:
      - /srv/autostorager:/incoming
```

```bash
docker compose up -d
mkdir -p /srv/autostorager
```

Then enter `/incoming` as the folder on the module's page, not `/srv/autostorager`.
The module creates `successful/` and `failed/` inside that folder itself, so the
host sees them under `/srv/autostorager/successful` and `/srv/autostorager/failed`.

On Windows and macOS the server is a normal process and the folder is read
directly, so no mount is involved.
