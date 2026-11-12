(content) =>
<>
    {content}
    <ul>
    {pages({ path: "docs/**", where: { tags: page.slug } }).map(doc => (
        <li>
        <a href={doc.path}>{doc.title}</a>
        </li>
    ))}
    </ul>
</>
