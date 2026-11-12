---
title: "Code highlighting"
kind: manual
num: 22
order: 22
part: "shipping"
---

Fenced code blocks in Markdown with a language are highlighted using CSS
classes, so your stylesheet controls how code looks:

````markdown
```go
func main() {}
```
````

renders as:

```html
<pre class="chroma" tabindex="0"><code class="language-go"><span class="line"><span class="cl"><span class="kd">func</span> <span class="nf">main</span><span class="p">()</span> <span class="p">{}</span>
</span></span></code></pre>
```

- The `<pre>` has `class="chroma"` and no inline styles. `tabindex="0"` lets
  keyboard users scroll wide blocks.
- The `<code>` has `class="language-<lang>"`, using the first word of the fence
  as written.
- Each line is `<span class="line"><span class="cl">...</span></span>`, and
  each token is a `<span>` with Chroma's short class name for its type. Plain
  text gets no span.

Common classes:

<div class="tw"><table>
<thead><tr><th>Class</th><th>Token</th></tr></thead>
<tbody>
<tr><td><code>k</code></td><td class="prose">Keyword</td></tr>
<tr><td><code>kd</code></td><td class="prose">Declaration keyword (<code>func</code>, <code>var</code>)</td></tr>
<tr><td><code>nf</code></td><td class="prose">Function name</td></tr>
<tr><td><code>nx</code></td><td class="prose">Other name</td></tr>
<tr><td><code>s</code>, <code>s1</code>, <code>s2</code></td><td class="prose">Strings</td></tr>
<tr><td><code>m</code>, <code>mi</code></td><td class="prose">Numbers</td></tr>
<tr><td><code>c</code>, <code>c1</code>, <code>cm</code></td><td class="prose">Comments</td></tr>
<tr><td><code>o</code></td><td class="prose">Operator</td></tr>
<tr><td><code>p</code></td><td class="prose">Punctuation</td></tr>
</tbody>
</table></div>

The full list is in Chroma's token type table
(<https://github.com/alecthomas/chroma/blob/master/types.go>). The markup is
the same as Hugo's, so any Chroma stylesheet works: generate one with Chroma's
CLI or `hugo gen chromastyles` and paste it into your CSS.

Taragen adds no highlighting CSS. Without any, code shows in your normal `pre`
and `code` styles. A minimal monochrome style:

```css
.chroma [class^="k"] { font-weight: bold; }                /* keywords */
.chroma [class^="c"]:not(.cl) { font-style: italic; }      /* comments */
```

Dark mode is just your usual `prefers-color-scheme` rules.

Blocks in a language Chroma doesn't know, blocks without a language, indented
code blocks, and inline code are plain `<pre><code>` (keeping
`class="language-<lang>"` when a language was given). Taragen never guesses a
language.
