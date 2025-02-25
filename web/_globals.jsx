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