// The plain text version of each page: its title, then its source.
data({ contentType: "text/plain; charset=utf-8" });
`${page.title ?? page.path}\n\n${page.src}`
