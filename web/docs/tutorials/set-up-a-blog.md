---
title: "Set up a blog"
kind: tutorials
num: 1
order: 1
aliases: [/docs/tutorials/setting-up-a-blog, /docs/tutorials/rss-feed]
uses: [/docs/manual/drafts, /docs/manual/non-html-pages-and-feeds]
---

Posts are ordinary pages. Grouping them in year folders keeps the tree tidy and
gives you a natural archive.

1. Create an `articles/` folder with a folder for the current year, and a post
   in it, `articles/2025/hello-world.md`:

   ```markdown
   ---
   title: Hello, world
   date: 2025-01-01
   description: The first post.
   ---

   This is my first post.
   ```

   The `date` orders posts in listings, and a date in the future keeps a post
   out of `build` until that day (see [Drafts](/docs/manual/drafts)).

2. Give posts a frame with `articles/_layout.jsx`. It wraps every page in
   `articles/`, and the root layout wraps the result:

   ```jsx
   (content) =>
   <article class="post">
     <h1>{page.title}</h1>
     <time>{page.date}</time>
     {content}
   </article>
   ```

3. Add the blog's front page, `articles/index.jsx`, listing posts by year,
   newest first. Year folders have no index file of their own, so they're
   grouping nodes with `isDir` set and their posts in `subpages`:

   ```jsx
   data({ title: "Articles", layout: "_layout" });
   <h1>Articles</h1>
   {pages("articles").filter(p => p.isDir).reverse().map(year =>
     <section>
       <h2>{year.slug}</h2>
       <ul>
         {year.subpages.slice().reverse().map(post =>
           <li><time>{post.date}</time> <a href={post.path}>{post.title}</a></li>
         )}
       </ul>
     </section>
   )}
   ```

   Setting `layout: "_layout"` uses only the root layout for this page, since
   the post layout would add a stray date heading.

   For a flat list instead, one call reaches every post however it's nested:

   ```jsx
   {pages("articles/**").reverse().map(post => partial("_partials/card", post))}
   ```

4. Add a feed. `articles/feed.xml.jsx` builds `/articles/feed.xml`; the full
   template is in [Non-HTML pages and feeds](/docs/manual/non-html-pages-and-feeds). Set
   `siteURL` in your root `_defaults.jsx` so the feed can build absolute links,
   and advertise the feed from the root layout's `<head>`:

   ```jsx
   <link rel="alternate" type="application/rss+xml" title="Articles" href="/articles/feed.xml" />
   ```

5. Run `taragen serve` and open `/articles`.
