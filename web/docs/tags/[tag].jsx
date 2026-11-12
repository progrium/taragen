params(() => pages("docs/**").flatMap(doc => doc.tags ?? []).map(tag => ({ tag })));

data({
    title: `Tag: ${page.params.tag}`,
});

<p>Docs tagged <code>{page.params.tag}</code>:</p>
