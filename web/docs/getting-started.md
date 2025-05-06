---
title: Getting Started
---

## Install

To install with Homebrew, run:
```
brew tap progrium/homebrew-taps
brew install taragen
```

TODO: add instructions for symlink?

## Overview

* Pages can be .jsx files or .md files
* Any filename that begins with an underscore will be hidden. For example, _layout.jsx, _globals.jsx, _partials.jsx

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
A single page can either go in a directory or not. About/index.jsx or about.jsx will resolve to /about. You can do this for posts as well, which can be useful if you have several files that would be grouped with a single blog post.

Note: .jsx files MUST have a single outer element. Most of the time Taragen can detect this and add it for you, but if you run into any odd parsing errors that might be why. 

## Layouts
Taragen is designed to use nested layouts. That is, a page will use the ```_layout.jsx``` file that's in the same directory, and that layout will be nested inside any layouts that are in parent directories.
We recommend having a single base ```_layout.jsx``` in your root folder that has a global header and footer, and individual layouts for various site sections as needed.

### Create a Layout
In this example, we’re going to make a layout for a specific page. We have an ```/about``` directory that contains an ```index.jsx```. we’ll add a ```_layout.jsx``` file to the ```/about``` directory. 

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
If you want to override the default behavior, simply specify the layout in the markdown metadata like so:

```
---
layout: about/_layout
___
```
```
data({
    layout: "about/_layout"
});
```

NOTE: layout references should be referenced relative to root. So if you want to reference the root level ```_layout```, you would use ```_layout```.

## Partials
Partials are reusable HTML that can be referenced in your pages or layouts. 

Partials are designed to take arguments, so even if you have no arguments you should begin your partial like so:
```
() =>
```

### Example: Adding Navigation
In a top-level /partials directory, add a ```_nav.jsx``` file with:

```
() =>
<ul>
	<li><a href="/">Home</a></li>
	<li><a href="/blog">Blog</a></li>
	<li><a href="/about">About</a></li>
</ul>
```


To reference the partial:
`{{ safeHTML `{{ partial "_nav" }}` }}` for Markdown, 
or ``` {partial("_nav")}``` for jsx.

## Globals
The ```_globals.jsx``` file is javascript that is available everywhere. You can put variables, helper functions, etc. in it.

### Globals Examples
Create a ```_globals.jsx``` file with the following content:
```
siteName = "My Site"
```
In your index.jsx file, add the siteName variable. The value in globals will be used for the variable.
```
<html>
    <body>
        <h1>{siteName}</h1>
    </body>
</html>
```
Note: Because it’s a jsx file, you can also create HTML element variables like ```siteName = <h4>My Site</h4>```

## Helpers

### Page
things like page, are there others?

* pages(”blog/*”) will give you all of the pages inside the direct children of blog

list attributes like page.slug, page.path, page.subpages

### Drafts
A file can be marked as a draft by putting draft: true in the metadata.

When you run serve (run locally), it shows drafts. When you use build (deploy for production), drafts are not generated.