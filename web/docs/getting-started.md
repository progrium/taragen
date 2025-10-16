---
title: Getting Started
---

## Install

To install with Homebrew, run:
```
brew tap progrium/homebrew-taps
brew install taragen
```

## Overview

* Pages can be JSX files or Markdown files, optionally with page data frontmatter
* Partials are JSX files for reusable components you can include in other JSX
* Layouts are special partials that wrap pages according to the file hierarchy
* Globals are JS or JSX values (helper functions, variables) available in all JSX



## Hello World

Create a directory for your site. From that directory, create an index.jsx file with simple HTML:

```
<html>
    <body>
        <h1>Hello World</h1>
    </body>
</html>
```

Then run:
```
taragen serve
```

## Adding Pages

A page can either be a file or a directory with an index file. Both `about/index.jsx` and `about.jsx` will resolve to `/about`. For a blog, posts are just pages, and using a directory with index for posts can be useful if you have several related files for the post.

Note: JSX files should try to have a single outer element. Multiple outer elements support is only partially supported, so if you run into any odd parsing errors that might be why. 

## Layouts

Layouts are partials that take a single `content` argument and by convention live in `_layout.jsx` files. Any directory with a `_layout.jsx` file will wrap that layout around any pages in this directory and subdirectories. Directory layouts nest, so any `_layout.jsx` files in a parent directory will apply to pages after any layout in the current directory. It is very common to use a single base `_layout.jsx` in your root folder for a global header and footer, and then nested directory layouts for site sections as needed.

### Create a Layout

In this example, we’re going to make a layout for a specific page. We have an `/about` directory with an `index.jsx`. We’ll add a `_layout.jsx` file to the `/about` directory. Since a layout is technically a special partial, the file should define a JavaScript function like partials. However, there will always only be the one `content` argument.

```
(content) => {
	title = "About";
	return (
		<html>
			<body class="about" title={title}>
				{content}
			</body>
		</html>
	)
}
```

### Override Default Layout Behavior

You can prevent a page from having the directory layout applied, as well as explicitly set the layout for a page, by specifying a `layout` field in the page data. An empty value means no layout. Any other value should be a path to a layout partial file relative to the root and without file extension.

```
---
layout: about/_alternative_layout
---
Markdown content.
```

The `layout` data field can also be set in layouts themselves using `data({ ... });` as the first statement of the layout. It must end with the semi-colon.

```
data({
    layout: "about/_alternative_layout"
});
(content) => {
	// ...
}
```

## Partials

Partials are functions that return reusable JSX and can be called in pages, layouts, and other partials. Any JSX. Partials are defined in partial files, which are typically named prefixed with an underscore so they are ignored as pages.

Since partials are functions, even if you have no arguments you still need the minimal syntax to define a function by beginning the partial file like so:
```
() =>
```

Note: The partial file doesn't need to start with an underscore if it is in a directory ignored for pages using the same underscore prefix.

### Example: Adding Navigation

In a top-level `/_partials` directory, add a `nav.jsx` file with:

```
() =>
<ul>
	<li><a href="/">Home</a></li>
	<li><a href="/blog">Blog</a></li>
	<li><a href="/about">About</a></li>
</ul>
```

To reference the partial in JSX:

```
 {partial("_partials/nav")}
```

Any extra arguments to the call to `partial` will be passed to the partial function.

You can also use partials in Markdown using Go template syntax:

```
[[ "[[ partial \"_partial/nav\" ]]" ]]
```

## Globals

A file called `_globals.jsx` can define variables and helper functions that will be available in all JSX files. 

Typically there is just one globals file in the root, but any directory with a `_globals.jsx` file will override global values for the directory and all subdirectories. Nested global files are merged, so if you wish to make a global defined in a parent directory unavailable, you would set it to `undefined` in the inner globals file.

### Globals Examples

Create a `_globals.jsx` file with the following content:

```
siteName = "My Site"
```

In your `index.jsx` file, use the `siteName` variable. It will have the value set in the globals file.

```
<html>
    <body>
        <h1>{siteName}</h1>
    </body>
</html>
```

Note: Because it’s a JSX file, you can use JSX/HTML for variable values like `siteName = <h4>My Site</h4>`. 

### Page
things like page helper, are there others?

* pages(”blog/*”) will give you all of the pages inside the direct children of blog

list attributes like page.slug, page.path, page.subpages

### Drafts

A page can be marked as a draft by setting `draft` to `true` in the page data. When you run locally with serve, it shows drafts. When you use build (for production), draft pages are not generated.

