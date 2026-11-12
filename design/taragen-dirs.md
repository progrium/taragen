---
slug: taragen-dirs
title: Special files in .taragen directories
status: draft
created: 2026-11-12
authors: [progrium]
depends: []
areas: [site, jsx, serve]
branch: []
---

# Proposal: Special files in .taragen directories

## Abstract

Let any directory of a site hold a `.taragen/` folder whose files are the same special files taragen reads today, named without the `_` prefix: `.taragen/layout.jsx`, `.taragen/globals.jsx`, `.taragen/defaults.jsx`, `.taragen/outputs/`, and, at the root, `.taragen/redirects`.
Every other file in a `.taragen/` folder is a partial, found by name from the page's directory upward, so `partial("footer")` finds the nearest `.taragen/footer.jsx`.
The `_` forms keep working unchanged, and both can be mixed in one site: a `.taragen/` folder where a directory has several special files, a bare `_layout.jsx` where it has one.
A directory that has both forms of the same file fails the build.

## Background

A site's special files sit among its pages, marked by a `_` prefix (see Site and file conventions in the [spec](SPEC.md)).
That keeps the file tree as the configuration, but a site root collects many of them.
The example site in `web/` has six `_` entries at its root next to two pages, and the progrium.xyz redesign has seven:

```
web/
├── _assets/
├── _blog/
├── _globals.jsx
├── _layout.jsx
├── _outputs/
├── _partials/
├── _redirects
├── docs/
└── index.jsx
```

The `_` prefix also does a second job: it is the only way to leave a file or folder out of the site, such as the parked `web/_blog/` section and the unfinished `web/docs/_overview.md`.
A `.` prefix leaves things out too, and the walker already prunes dot directories, but nothing uses it for site content today.

Every special file is found the same way: for each directory from the page's own up to the root, taragen checks `path.Join(dir, "_layout.jsx")` and keeps the matches, nearest first or root first depending on the file (`loadGlobals`, `loadDefaults`, and `Render` in `page.go`, and `outputTemplates` in `outputs.go`).
Partials and explicit `layout` values are different: they are paths from the site root, such as `partial("_partials/footer")` and `layout: "_blog/_layout"`, so a section can't swap in its own header without every caller naming it.

Other tools keep their configuration in a dot folder at the place it applies to: `.github/`, `.vscode/`, `.cargo/`.
Taragen can do the same per directory, so a root's worth of `_` entries becomes one folder, without giving up the per-directory scoping that makes layouts and defaults work.

## Proposal

### The .taragen folder

Any directory of a site, including the root, may contain a folder named exactly `.taragen`.
Its files take the place of the `_` special files in that directory:

| `.taragen/` file | Same as | Applies to |
| --- | --- | --- |
| `layout.jsx` / `layout.tmpl` | `_layout.jsx` / `_layout.tmpl` | Pages in this directory and below |
| `globals.jsx` | `_globals.jsx` | JSX in this directory and below |
| `defaults.jsx` | `_defaults.jsx` | Pages in this directory and below |
| `outputs/<name>.jsx` | `_outputs/<name>.jsx` | HTML pages in this directory and below |
| `redirects` (root only) | `_redirects` (root only) | The site |
| any other `.jsx` / `.tmpl` file, at any depth | a partial | Pages in this directory and below, by name |

These are the reserved names.
Everything about each file is as it is today: what it exports or returns, the order files along the path are applied in, and nearest-wins for layouts and outputs.
Only where taragen looks changes: for each directory on the walk from a page up to the root, it checks both `dir/_name` and `dir/.taragen/name`.

Inside a `.taragen/` folder:

- Names starting with `.` are ignored, as everywhere, so editor swap and backup files don't matter.
- Names starting with `_` are an error naming the file, so a half-converted `.taragen/_layout.jsx` doesn't silently become a partial.
- A `redirects` file anywhere but the root `.taragen/` is an error naming it.
- Files other than page sources and `redirects` are an error naming the file, since nothing would read them.

Nothing in a `.taragen/` folder is a page, a static file, or copied into `_public/`.
The walker already prunes dot directories, so `pages()`, the route table, and `build` don't see them.

### Mixing forms

A site may use `.taragen/` folders in some directories and `_` files in others, and a directory may have a `.taragen/` folder and `_` files for different names.
The lookup treats `dir/_layout.jsx` and `dir/.taragen/layout.jsx` as the same slot for that directory, so mixing directories changes nothing about the order things apply in.

A directory with both forms of one slot fails the build, naming both files:

- `_layout.*` and `.taragen/layout.*`
- `_globals.jsx` and `.taragen/globals.jsx`
- `_defaults.jsx` and `.taragen/defaults.jsx`
- `_outputs/` and `.taragen/outputs/`, as whole folders
- at the root, `_redirects` and `.taragen/redirects`

The check runs when the site is parsed, so `serve` reports it too.

### Partial names

`partial(name)` in JSX and `[[ partial "name" ]]` in templates resolve `name` in one of two ways:

- **A name with a `_`-prefixed segment**, such as `_partials/footer` or `docs/_card`, is a path from the site root, exactly as today.
- **Any other name**, such as `footer` or `partials/footer`, is looked up as `.taragen/<name>.jsx`, then `.taragen/<name>.tmpl`, in each directory from the page's directory up to the root, nearest first. If no `.taragen/` folder has it, it falls back to a path from the site root, as today.

The page's directory is the directory of the page being rendered, also when the call is in a layout or another partial, so a section's `.taragen/header.jsx` replaces the root one for every page in that section, including inside the root layout.
A generated page looks along its generated path, as layouts and defaults already do.

A name may use `/` to reach into subfolders of `.taragen/`, so a site that keeps a `partials/` subfolder calls `partial("partials/footer")`, and one that doesn't calls `partial("footer")`.
A name with an empty, `.`, or `..` segment, or a leading `/`, is an error naming it.

### The layout key

A `layout` value in page data resolves the same way as a partial name.
`layout: "wide"` uses the nearest `.taragen/wide.jsx`, and `layout: "_blog/_layout"` is a root path as today.
`layout: ""` still turns layouts off.

### What does not change

- `_assets/` keeps its meaning, and there is no `.taragen/assets/`. Assets are content that people edit and look for, and a plain `assets/` folder already works as static files for everything but `.md`, `.jsx`, and `.tmpl` files.
- The build output stays at `_public/`.
- `_` keeps its meaning everywhere else: `_`-prefixed files and folders are still special, never pages, and never copied, so parked sections like `web/_blog/` keep working. A `.`-prefixed name leaves content out the same way, and the manual says so.
- Generated page parameter values still can't start with `.` or `_`.
- A `.taragen/` folder inside a bracketed directory is an error, as `_` files are there.

### Live reload

`serve` ignores changes under dot-prefixed paths today.
It treats a segment named exactly `.taragen` as an ordinary one, so a change under `.taragen/` is classified like any other path: page sources and directories reparse, and `redirects`, which has no extension, reparses too.
Other dot segments, including ones inside `.taragen/`, stay ignored.

### Example trees

The example site in `web/`, with the root converted and sections keeping their single `_layout.jsx`:

```
web/
├── .taragen/
│   ├── globals.jsx
│   ├── layout.jsx
│   ├── redirects
│   ├── outputs/
│   │   ├── card.png.jsx
│   │   └── txt.jsx
│   └── partials/
│       ├── footer.jsx
│       └── header.jsx
├── _assets/
├── _blog/
├── docs/
│   ├── _layout.jsx
│   ├── _overview.md
│   ├── getting-started.md
│   ├── index.jsx
│   ├── tags/
│   └── tutorials/
└── index.jsx
```

The progrium.xyz redesign, where `logs/` has enough special files to earn a folder:

```
progrium.xyz/
├── .taragen/
│   ├── defaults.jsx
│   ├── globals.jsx
│   ├── layout.jsx
│   ├── redirects
│   ├── card.jsx
│   ├── footer.jsx
│   ├── header.jsx
│   └── outputs/
│       ├── card.png.jsx
│       └── txt.jsx
├── assets/
├── about.md
├── index.jsx
├── articles/
│   ├── _defaults.jsx
│   ├── _layout.jsx
│   ├── feed.xml.jsx
│   ├── index.jsx
│   └── 2021/
├── logs/
│   ├── .taragen/
│   │   ├── layout.jsx
│   │   └── outputs/
│   │       ├── card.png.jsx
│   │       └── txt.jsx
│   ├── [year]/
│   └── 2026/
├── projects/
├── docs/
└── tags/
```

## Rationale

**Replacing `_` instead of adding to it.**
Dropping the `_` forms would make `.taragen/` the only way, free `_` names for host files like Netlify's `_headers`, and keep one spelling in the docs.
It would also break every existing site, and it would force a one-file hidden folder on every section that only has a layout, where a bare `_layout.jsx` is clearer.
Both forms cost a second lookup per directory and two spellings in the docs; the lookup is one `stat` per slot, and the manual can lead with `.taragen/` and describe `_` files as the inline form.

**One root `.taragen/` mirroring the tree.**
A single folder like `.taragen/articles/layout.jsx` (close to Hugo's `layouts/`) moves all configuration out of the content tree.
It separates a section's layout from the section, so moving or deleting a section means editing two places, and it loses the "look in this folder to see what applies here" property.

**`.taragen/assets/` and `.taragen/public/`.**
Assets are files people edit and look for, and output is something deploy tools point at.
Hiding either costs more than it saves: some upload tools skip hidden paths by default (`actions/upload-artifact` does), and the root only gains one entry.

**Partial lookup from the page instead of the calling file.**
Resolving from the calling file's directory would make a root layout's `partial("header")` always find the root header.
Resolving from the page lets a section override a partial for all its pages without its own layout, the same way it can already override defaults, and matches how layouts are found.

**Keeping the root-path fallback for partial names.**
Without it, `partial("blog/card")` pointing at a real `blog/card.jsx` would stop working.
With it, every existing call resolves as before, since no site has `.taragen/` folders yet.

**The name `.taragen`.**
It names the tool, so it can't be mistaken for content or another tool's folder, and it follows `.github` and `.vscode`.

**Hidden folders are easy to miss.**
`ls` and Finder hide them, and ripgrep and fd skip them unless given `--hidden`, so searching a site for `header` won't find `.taragen/header.jsx` by default.
Editors and git show them.
The manual notes this where it introduces `.taragen/`; it is the main cost of the form, and a reason to keep `_` files.

## Compatibility

- **Site conventions:** additive. A `.taragen` folder is ignored today, so a site that already has one, which is unlikely, would start reading it. A directory with both forms of a slot can't exist in a working site today, so the new error only applies to new layouts.
- **JSX runtime and templates:** `partial` and `layout` names without a `_` segment look in `.taragen/` folders first. This changes the result only for a site that has a `.taragen/` file with the same name as a root path it already includes.
- **Serve mode:** changes under `.taragen/` now reparse and reload.
- **CLI, page data, Markdown, and distribution:** unchanged.

## Security

`.taragen/` folders are inside the site directory, which taragen already reads in full.
Partial and layout lookups stop at the site root, and names with `.` or `..` segments or a leading `/` are rejected, so a name can't reach outside it.
Nothing in a `.taragen/` folder is copied into `_public/`, so it can't publish configuration that `_` files keep private today.

## Done criteria

- [ ] `TestTaragenDirLayouts`: a root `.taragen/layout.jsx` and a `docs/.taragen/layout.jsx` wrap a docs page nearest first, as the `_` forms do.
- [ ] `TestTaragenDirMixed`: a root `.taragen/layout.jsx` with `docs/_layout.jsx`, and a root `_layout.jsx` with `docs/.taragen/layout.jsx`, both produce the same output as the all-`_` site.
- [ ] `TestTaragenDirGlobalsDefaults`: `.taragen/globals.jsx` and `.taragen/defaults.jsx` apply root first, and a nested one overrides a root `_` one.
- [ ] `TestTaragenDirOutputs`: `.taragen/outputs/txt.jsx` builds `.txt` outputs, and `logs/.taragen/outputs/card.png.jsx` replaces a root `_outputs/card.png.jsx` for logs.
- [ ] `TestTaragenDirConflicts`: each slot with both forms in one directory fails the build naming both files.
- [ ] `TestTaragenDirPartials`: `partial("footer")` finds `.taragen/footer.jsx`; `partial("partials/footer")` finds `.taragen/partials/footer.jsx`; a `docs/.taragen/header.jsx` replaces the root one for docs pages, including when called from the root layout; `partial("_partials/footer")` and a root-path name with no `.taragen/` match work as before.
- [ ] `TestTaragenDirPartialNames`: `partial("../x")`, `partial("./x")`, `partial("/x")`, and `partial("a//b")` fail naming the partial.
- [ ] `TestTaragenDirLayoutKey`: `layout: "wide"` uses the nearest `.taragen/wide.jsx`, and `layout: "_blog/_layout"` works as before.
- [ ] `TestTaragenDirRedirects`: a root `.taragen/redirects` is read and written to `_public/_redirects`; a nested one fails; both root forms together fail.
- [ ] `TestTaragenDirBadFiles`: `.taragen/_layout.jsx`, `.taragen/notes.txt`, and a `.taragen/` folder inside `tags/[tag]/` each fail naming the file; `.taragen/.layout.jsx.swp` is ignored.
- [ ] `TestTaragenDirNotPublished`: `build` writes nothing from `.taragen/` folders into `_public/`, and `pages()` returns nothing from them.
- [ ] `TestClassifyChangeTaragen`: `.taragen/layout.jsx` and `.taragen/redirects` reparse and reload; `.taragen/.layout.jsx.swp` and `.git/HEAD` are ignored.
- [ ] The `web/` site moves its root special files into `.taragen/`, and `make web-build` output is byte-identical to the build before the move.
- [ ] The spec and MANUAL.md describe `.taragen/` folders, the conflict rule, partial and layout name lookup, and the hidden-folder caveat.

## Implementation

1. Add one lookup that, for a directory and a slot, returns the `_` path, the `.taragen/` path, or a conflict error, and use it in `loadGlobals`, `loadDefaults`, `Render`, and `outputTemplates`.
2. Read the root redirects file from either location, with the conflict and nested-file errors.
3. Validate `.taragen/` folders during the walk: the reserved names, the `_` and unknown-file errors, and the bracketed-directory error.
4. Resolve partial and `layout` names through `.taragen/` folders from the page's directory, with the root-path fallback and the name checks, in `Site.Partial` and the layout loop in `Render`.
5. Let `classifyChange` treat a `.taragen` segment as ordinary.
6. Convert the root of the `web/` site, then update the spec and MANUAL.md.

## Open issues

- Should there be a `taragen migrate` command that moves a directory's `_` files into `.taragen/` and rewrites `partial` and `layout` names? Moving files is simple; rewriting names inside JSX and templates is the hard part.
- Should a `.taragen/` folder be allowed in a bracketed directory, applying to every page generated under it? This proposal keeps the current rule for `_` files, which makes it an error.
- Should the manual and the starter examples lead with `.taragen/`, or present the two forms as equals?
