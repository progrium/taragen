---
title: Setting up a blog
---

Create a blog directory in your root folder.

Inside the blog directory, add a `_layout.jsx` file.

We recommend grouping posts by date, so create a directory inside blog for the current year.

Inside the current year, create a file called `post-title.md` with the following format:


```
---
title: "Your Title"
date: 2025-01-01
---

content
```


	
Inside the `blog/_layout.jsx` file, add the following:

```
(content) =>
    <div class="blog">
    <main>
        <h1>{page.title}</h1>
        {content}
    </main>
    </div>
```
	
Inside `/blog`, create an `index.jsx` file for the homepage. Add the following:

*Note the empty tags, because JSX requires a single outer element.*

```
{{ safeHTML `<>
	<h1>Blog</h1>
	{pages("blog").filter(page => page.isDir).reverse().map(year => (
		<div>
			<h3>{ year.slug }</h3>
			<ul class="mb-8">
			{year.subpages.map(post => (
			<li class="flex space-x-2 my-2">
				<span class="text-gray-400 w-24">{post.date}</span>
				<a class="underline" href={post.path}>{post.title}</a>
			</li>
			))}
			</ul>
		</div>
	))}
</> ` }}
```
