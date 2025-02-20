data({
    title: "Homepage"
});
<main>
<h1>Taragen</h1>
<ul>
{pages(".").map(page => <li><a href={page.path}>{page.title}</a></li>)}
</ul>

</main>
