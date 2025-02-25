data({
    title: "Homepage"
});
<main>
<h1>Taragen</h1>
<ul>
{pages(".").filter(page => page.slug != ".").map(page => 
    (page.isDir) 
        ? <li><a href={page.path}>{toTitleCase(page.slug)}</a></li>
        : <li><a href={page.path}>{page.title}</a></li>
)}
</ul>

</main>
