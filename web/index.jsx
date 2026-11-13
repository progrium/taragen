data({
  title: "taragen",
  description: "Taragen is a fast, minimal static site generator in a single binary. Write pages in JSX, Markdown or Go templates; the folder is the configuration.",
});
<div class="home">
  <div class="hero">
    <h1>Your folder is the website.</h1>
    <p>Taragen is a fast, minimal static site generator in a single binary. Write pages in JSX, Markdown or Go templates, arrange them in folders, and taragen turns the folder into a site. No config file, no Node.js, no <code>node_modules</code>.</p>
  </div>
  <div class="install">
<pre class="chroma" data-lang="sh">{raw(`<span class="p">$</span> brew tap progrium/homebrew-taps
<span class="p">$</span> brew install taragen
<span class="p">$</span> taragen serve   <span class="c1"># http://localhost:8088, live reload</span>`)}</pre>
  </div>
  <div class="keys home-keys">
    <a class="key solid" href="/docs/manual/quick-start">quick start ⏎</a>
    <a class="key" href="/docs">read the manual</a>
    <a class="key" href="/docs/tutorials">tutorials</a>
    <span class="m soft">or grab a binary from <a href="https://github.com/progrium/taragen/releases">releases</a></span>
  </div>

  <h2 class="sec">how it works</h2>
  <div class="treemap">
<pre class="chroma tree">{raw(`mysite/
  _layout.jsx        <span class="c1">frame for every page</span>
  index.jsx          <span class="p">→ /</span>
  about.md           <span class="p">→ /about</span>
  blog/
    hello.md         <span class="p">→ /blog/hello</span>
    feed.xml.jsx     <span class="p">→ /blog/feed.xml</span>
    [tag].jsx        <span class="p">→ /blog/go, /blog/plan9 …</span>
  _outputs/
    txt.jsx          <span class="p">→ /about.txt, /blog/hello.txt …</span>`)}</pre>
    <div>
      <p>Every file is a page and every folder is a section. Underscored files are the machinery: <a href="/docs/manual/layouts">layouts</a> wrap pages, <a href="/docs/manual/partials">partials</a> are components, and <a href="/docs/manual/extra-outputs-per-page">outputs</a> add a <code>.txt</code> or a share card beside each page.</p>
      <p>JSX runs in a JavaScript engine inside the binary and renders to plain HTML. Nothing is shipped to the browser.</p>
    </div>
  </div>

  <h2 class="sec">why taragen</h2>
  <div class="feats">
    <div><h3>Write in the format that fits</h3><p>Markdown for writing, JSX for pages with logic, Go templates if you like them, all in one site.</p></div>
    <div><h3>Components without a framework</h3><p>Layouts and partials in JSX, rendered to HTML at build time.</p></div>
    <div><h3>Live reloading</h3><p><code>taragen serve</code> notices every change and refreshes the browser for you.</p></div>
    <div><h3>One binary</h3><p>Install it and you're done. Nothing to <code>npm install</code>, nothing to keep updated.</p></div>
  </div>
</div>
