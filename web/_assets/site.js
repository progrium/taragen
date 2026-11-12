// Dark mode keycap, and a language label and copy keycap on code blocks.
(function () {
  var root = document.documentElement;
  var key = document.getElementById("modekey");
  var label = document.getElementById("mk-l");
  function isDark() {
    var t = root.dataset.theme;
    return t ? t === "dark" : window.matchMedia("(prefers-color-scheme: dark)").matches;
  }
  function sync() { if (label) label.textContent = isDark() ? "light" : "dark"; }
  if (key) key.addEventListener("click", function () {
    var t = isDark() ? "light" : "dark";
    root.dataset.theme = t;
    try { localStorage.setItem("theme", t); } catch (e) {}
    sync();
  });
  sync();

  document.querySelectorAll("pre").forEach(function (pre) {
    if (pre.classList.contains("term") || pre.classList.contains("tree")) return;
    var code = pre.querySelector("code");
    var m = code && code.className.match(/language-(\S+)/);
    var lang = pre.dataset.lang || (m && m[1]) || "";
    var source = (code || pre).innerText.replace(/^\$ /gm, "");
    var bar = document.createElement("span");
    bar.className = "lang";
    if (lang) { var l = document.createElement("span"); l.textContent = lang; bar.appendChild(l); }
    if (navigator.clipboard) {
      var b = document.createElement("button");
      b.type = "button"; b.className = "key sm"; b.textContent = "copy";
      b.addEventListener("click", function () {
        navigator.clipboard.writeText(source).then(function () {
          b.textContent = "copied";
          setTimeout(function () { b.textContent = "copy"; }, 1200);
        }, function () {});
      });
      bar.appendChild(b);
    }
    if (bar.childNodes.length) { pre.appendChild(bar); pre.classList.add("has-lang"); }
  });
})();
