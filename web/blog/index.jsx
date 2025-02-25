<>
<h1>Blog</h1>
{pages("blog").filter(page => page.isDir).reverse().map(year => (
    <div>
      <h3>{ year.slug }</h3>
      <ul class="mb-8">
      {year.subpages.map(post => (
        <li class="flex space-x-2 my-2">
          <span class="text-gray-400 w-24">{post.date}</span>
          <a class="underline" href={post.path}>{post.title}</a>
        </li>
      ))}
      </ul>
    </div>
  ))}
</>