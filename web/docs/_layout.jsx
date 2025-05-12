(content) =>
<>
        <nav class="sidebar">
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

        <section>
            <h1>{page.title}</h1>
            {content}
        </section>

</>
    