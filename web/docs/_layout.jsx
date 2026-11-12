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
            {page.backlinks.length > 0 &&
                <aside class="backlinks">
                    <h2>Linked from</h2>
                    <ul>
                        {page.backlinks.map(ref => (
                            <li><a href={ref.path}>{ref.title || ref.path}</a></li>
                        ))}
                    </ul>
                </aside>
            }
        </section>

</>
    