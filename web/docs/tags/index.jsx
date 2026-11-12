data({
    title: "Tags",
});

<div>
    <ul>
    {pages("docs/tags").map(tag => (
        <li>
        <a href={tag.path}>{tag.title}</a>
        </li>
    ))}
    </ul>
</div>
