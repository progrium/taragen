(content) =>
<div>
<nav>
    <ul>
    {pages("docs").map(page => (
        <li>
            <a href={page.path}>{page.title}</a>
            {page.isDir && 
                <ul>
                    {pages(page.path).map(sub => (
                        <li>
                            <a href={sub.path}>{sub.title}</a>
                        </li>
                    ))}
                </ul>
            }
        </li>
    ))}
    </ul>
</nav>
<main>
    {content}
</main>
</div>
    