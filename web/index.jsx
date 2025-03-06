data({
    title: "Homepage"
});
<main>
<ul>
{pages(".").filter(page => page.slug != ".").map(page => 
    (page.isDir) 
        ? <li><a href={page.path}>{toTitleCase(page.slug)}</a></li>
        : <li><a href={page.path}>{page.title}</a></li>
)}
</ul>

</main>
