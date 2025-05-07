data({
    title: "Homepage"
});
<main>

 <section>
    {/* banner image goes here */}
 </section>

 <section id="feature-list">
    <div>
        <p class="h3">Multi-format support</p>
        <p>We’re flexible. Write your content in Markdown, JSX, or even Go templates.</p>
    </div>
    <div>
        <p class="h3">Component-based templates</p>
        <p>Don’t repeat yourself. JSX allows for a component-driven approach to building pages.</p>
    </div>
    <div>
        <p class="h3">Live reloading</p>
        <p>Site development made easy. We detect changes and automatically refresh your site. </p>
    </div>
 </section>

<section>
    <h2>Getting Started</h2>
    <h3>Install</h3>
    <p>To install with Homebrew, run:</p>
    <pre><code>brew tap progrium/homebrew-taps
    brew install taragen
    </code></pre>

    <p><strong>TODO:</strong> add instructions for symlink?</p>

    <h3>Overview</h3>
    <ul>
        <li>Pages can be .jsx files or .md files</li>
        <li>Any filename that begins with an underscore will be hidden. For example, <code>_layout.jsx</code>, <code>_globals.jsx</code>, <code>_partials.jsx</code></li>
    </ul>

    <h3>Hello World</h3>
    <p>Create a directory for your site. From that directory, create an <code>index.jsx</code> file with simple HTML:</p>
    <pre><code>&lt;html&gt;
        &lt;body&gt;
            &lt;h1&gt;Hello World&lt;/h1&gt;
        &lt;/body&gt;
    &lt;/html&gt;
    </code></pre>
    <p>Then run:</p>
    <pre><code>taragen serve
    </code></pre>

    <h3>Further Exploration</h3>
    <p>Check out <a href="/docs">our docs</a> to see what Taragen can do.</p>
    
</section>


</main>
