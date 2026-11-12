siteTitle = "Taragen"
siteDescription = "Taragen is a JSX static site generator"
siteUrl = "https://taragen.xyz"

toTitleCase = (str) => {
    return str.replace(
      /\w\S*/g,
      function(txt) {
        return txt.charAt(0).toUpperCase() + txt.substr(1).toLowerCase();
      }
    );
}

// Greedy word wrap with the same measurements the share card renders with.
wrap = (text, font, width) => text.split(" ").reduce((lines, word) => {
  const last = lines[lines.length - 1];
  if (last && measureText(`${last} ${word}`, font) <= width) lines[lines.length - 1] = `${last} ${word}`;
  else lines.push(word);
  return lines;
}, []);
