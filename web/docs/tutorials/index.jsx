data({
    title: "Tutorials",
});

<div>
    <ul>
    {pages("docs/tutorials").map(page => (
        <li>
        <a href={page.path}>{page.title}</a>
        </li>
    ))}
    </ul>
</div>