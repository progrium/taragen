(content) =>
<section>
    <div class="row justify-center items-start sm:stack" style="gap: var(--16)">
        <nav class="flex: none; min-width: 256px;">
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

        <main class="grow">
            {content}
        </main>
    </div>
</section>
    