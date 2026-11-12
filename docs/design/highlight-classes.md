---
slug: highlight-classes
title: Code highlighting through CSS classes
status: implemented
created: 2026-11-12
authors: [progrium]
depends: []
areas: [markdown]
branch: [main]
---

# Proposal: Code highlighting through CSS classes

Nov 12, 2026 · Jeff Lindsay, h.d.a.

## Abstract

Highlighted code blocks in Markdown get CSS classes instead of inline `style` attributes, so the site's stylesheet decides how code looks.
Tokens are marked with Chroma's standard short class names (`k`, `nf`, `s`, ...) inside a `<pre class="chroma">`, which is the same markup Hugo produces, so existing Chroma stylesheets work unchanged.
Every fenced block with a language keeps a `language-<lang>` class on its `<code>`, whether or not Chroma knows the language.
Taragen emits no colors at all; a site with no highlighting CSS shows code in its own `pre` and `code` styles.

## Background

Markdown goes through goldmark with the `goldmark-highlighting` (Chroma) extension (see Markdown in the [spec](../spec.md)).
`format_md.go` enables it with its defaults, so the Chroma `github` style is hard-coded and every token is colored with an inline `style` attribute.
A fenced Go block renders today as:

```html
<pre tabindex="0" style="background-color:#fff;"><code><span style="display:flex;"><span><span style="color:#000;font-weight:bold">func</span> <span style="color:#900;font-weight:bold">main</span>() { ...
```

This has three problems:

- **The site can't change the look.** Inline styles beat any stylesheet rule short of `!important` on every span. There is no option to pick another style, and adding one would still give a single fixed palette.
- **It breaks dark mode.** The `pre` gets `background-color:#fff` and tokens get dark colors, whatever the page's color scheme. The progrium.xyz redesign is black and white with a dark mode and wants code in monochrome with bold keywords, which no fixed palette can give in both schemes.
- **It overrides the site's own `pre` styles.** The example site in `web/` styles `pre` with a brown background and white text, and the inline white background replaces it.

The output is also inconsistent: a block in an unknown language keeps `<code class="language-x">`, but a block Chroma highlights loses its language class.
Blocks with no language and inline code are not highlighted and carry no style; that stays the same.

## Proposal

### Markup

A fenced code block whose language Chroma recognizes renders as:

```html
<pre class="chroma"><code class="language-go"><span class="line"><span class="cl"><span class="kd">func</span> <span class="nf">main</span><span class="p">()</span> ...
</span></span></code></pre>
```

- The `<pre>` has `class="chroma"` and no `style` attribute.
- The `<code>` has `class="language-<lang>"`, using the info string's first word as written (`go`, `js`, `jsx`, ...).
- Each line is a `<span class="line">` wrapping a `<span class="cl">`, and each token is a `<span>` whose class is Chroma's short name for its token type, such as `k` (keyword), `kd` (declaration keyword), `nf` (function name), `s` (string), `c1` (comment), and `p` (punctuation). Tokens Chroma treats as plain text get no span.
- No element produced by highlighting has a `style` attribute.
- `tabindex="0"` stays on the `<pre>`, so keyboard users can scroll wide blocks.

Fenced blocks in a language Chroma doesn't know, fenced blocks without a language, indented code blocks, and inline code are unchanged: plain `<pre><code>` (with `class="language-<lang>"` when a language was given) and escaped text.

Taragen does not guess a language for blocks without one.

### Styling

Taragen ships no highlighting CSS and doesn't inject any into pages.
The class names are Chroma's and are the public contract; the spec lists the common ones and links Chroma's full token table.
A site styles them in its own stylesheet, for example this monochrome treatment for the redesign:

```css
.chroma [class^="k"] { font-weight: bold; }   /* every keyword type */
.chroma [class^="c"]:not(.cl) { font-style: italic; }   /* every comment type */
```

Because colors come from the stylesheet, a dark mode is the site's usual `prefers-color-scheme` rules and needs nothing from taragen.
A site that wants a ready palette can paste the CSS for any Chroma style, for example from Chroma's own CLI or Hugo's `hugo gen chromastyles`, since the class names are the same.

### Example site

The `web/` site gets a short highlighting section in `_assets/styles/style.css` that fits its brown `pre` background, so its docs keep colored code.

## Rationale

**Classes only, no switch.** Taragen has no site config file, and the filesystem is the config, so a "classes or inline" option would need a new place to live. Inline styles have no use a stylesheet can't cover, and the only sites using taragen are its own and progrium's, so a clean switch is cheap now and costly later.

**Chroma's names, not new ones.** Taragen could map tokens to readable names like `keyword` and `string`, but Chroma's short names are already documented, already used by Hugo and many themes, and come free from the formatter. Renaming them would add a mapping to maintain and cut sites off from existing stylesheets.

**No class prefix.** Chroma can prefix its classes (`chroma-k`) to avoid clashes, but the `.chroma` class on `<pre>` already scopes every rule, and prefixes would break compatibility with existing Chroma CSS.

**No built-in stylesheet generator.** A `taragen` command or a JSX global that prints the CSS for a named Chroma style would help sites that want a stock palette. It would add a CLI or runtime surface for something the Chroma and Hugo tools already do, and the redesign doesn't need it. It can be added later with its own proposal if it's missed.

**Client-side highlighting** (Prism, highlight.js) was rejected: it needs JavaScript on every page with code and flashes unstyled code, while the site already ships static HTML. The `language-<lang>` class still leaves that door open for a site that wants it.

**Highlighting in JSX pages** isn't covered. Code in JSX pages is written by hand today; a `highlight(code, lang)` global that returns the same markup would fit later and reuse this contract.

## Compatibility

- **Templates and Markdown:** the HTML for highlighted fenced blocks changes as described above. Sites that relied on the built-in `github` colors lose them until they add CSS. The text content and the block structure (`pre > code`) don't change, so selectors on `pre` and `code` keep working, and they now take effect where inline styles used to override them.
- **Example site:** `web/` gets highlighting CSS so its docs look the same or better.
- No changes to the CLI, site file conventions, page data, the JSX runtime, serve mode, or distribution.
- **Dependencies:** none added. The change uses options that `goldmark-highlighting` and Chroma (v0.10) already have. Moving to Chroma v2 would keep the same class names and isn't part of this proposal.

## Security

None. The change removes inline `style` attributes and adds fixed class names; token text is escaped by Chroma exactly as before, and the language name in `language-<lang>` is escaped as an attribute value as goldmark does today.

## Done criteria

- [x] `TestMarkdownHighlightClasses`: a fenced Go block renders with `<pre class="chroma">`, `<code class="language-go">`, and token spans with classes, and the output has no `style=` attribute.
- [x] `TestMarkdownHighlightUnknownLanguage`: a fenced block in an unknown language, a fenced block with no language, and an indented block render as plain `<pre><code>`, with `language-<lang>` only when a language was given.
- [x] `TestMarkdownHighlightEscapes`: a block containing `<`, `&`, and `"` is escaped inside token spans.
- [x] Demo: `taragen build` on `web/` has no `style=` attribute in any `pre` in `_public/`, and the docs pages show highlighted code with the site's CSS.
- [x] The spec's Markdown section describes the markup and the class contract, and links Chroma's token class list.

## Implementation

1. In `format_md.go`, configure the extension with `highlighting.WithFormatOptions(chromahtml.WithClasses(true))`, and a wrapper renderer (or Chroma pre wrapper) that writes `<pre tabindex="0" class="chroma"><code class="language-<lang>">`.
2. Add the three tests above.
3. Add highlighting rules to `web/_assets/styles/style.css`.
4. Update the Markdown section of the spec.

## Updates

2026-11-12: Implemented. The open issue was settled as drafted: the spec has a short table of common class names and links Chroma's full token table. Line spans keep Chroma's `hl` class when a block uses the `hl_lines` attribute. The example site's docs had no languages on their fences, so they were tagged (`jsx`, `html`, `css`, `sh`, `yaml`) to show highlighting.
