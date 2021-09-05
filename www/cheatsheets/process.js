// run as:
// deno run --allow-read --allow-write .\process.js

import { Marked } from "https://deno.land/x/markdown/mod.ts";
import hljs from "https://jspm.dev/highlight.js@11.0.1";
import { join, basename } from "https://deno.land/std@0.106.0/path/mod.ts";
import { DOMParser } from "https://deno.land/x/deno_dom/deno-dom-wasm.ts";
import { files } from "./processfiles.js";

function len(o) {
  if (o && o.length) {
      return o.length;
  }
  return 0;
}

function genHTML(innerHTML, fileName, meta) {
  let title = meta.title;
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8" />
  <title>${title} cheatsheet</title>
  <link href="s/main.css" rel="stylesheet" />
  <script src="s/main.js"></script>
</head>

<body onload="start()">
  <div class="breadcrumbs"><a href="/">Home</a> / <a href="index.html">cheatsheets</a> / ${title} cheatsheet</div>
  <div class="edit">
      <a href="https://github.com/kjk/blog/blob/master/www/cheatsheets/${fileName}" >edit</a>
  </div>
  ${innerHTML}
</body>  
</html>`
}

function genIndexHTML(files) {
  let innerHTML = "";
  for (let file of files) {
    innerHTML += `<div class="cslist-item"><a href="${file}.html">${file}</a></div>` + "\n";
  }
  const s = `<!DOCTYPE html>
<html>

<head>
  <meta charset="utf-8" />
  <title>cheatsheets</title>
  <link href="s/main.css" rel="stylesheet" />
  <script src="//unpkg.com/alpinejs" defer></script>
  <script src="s/main.js"></script>
</head>

<body onload="startIndex()">
  <div class="breadcrumbs"><a href="/">Home</a> / cheatsheets</div>

  <div x-init="$watch('search', val => { filterList(val);})" x-data="{ search: '' }" class="input-wrapper">
    <div><input @keyup.escape="search=''" id="search-input" type="text" x-model="search"></div>
  </div>

  <div class="cslist">
    ${innerHTML}
  </div>
</body>
</html>
`;
  const path = "index.html";
  Deno.writeTextFileSync(path, s)
}

function genTocHTML(toc) {
  let html = `<div class="toc">`;
  for (let e of toc) {
    if (len(e.a) === 0) {
      // handle a case where e.a is empty
      html += `\n<a href="#${e.id}">${e.name}</a><br>`
      continue;
    }
    let s = `\n<b>${e.name}</b>: `;
    let i = 0;
    for (let te of e.a) {
      if (i > 0) {
        s += ", ";
      }
      s += `<a href="#${te[1]}">${te[0]}</a>`;
      i++;
    }
    s += "<br>";
    html += s;
  }
  html += "</div>\n\n";
  return html;
}

function buildToc(s) {
  const doc = new DOMParser().parseFromString(s, "text/html");
  //console.log(doc.body);
  let toc = [];
  let curr = null;
  for (let node of doc.body.childNodes) {
    //console.log(node);
    let tag = node.tagName;
    if (!tag) {
      continue;
    }
    tag = tag.toLowerCase();
    const isHdr = tag == "h2" || tag == "h3";
    if (!isHdr) {
      continue;
    }
    const id = node.attributes["id"];
    const txt = node.textContent;
    if (tag === "h2") {
      curr = {
        name: txt,
        id: id,
        a: [],
      };
      toc.push(curr);
      continue;
    }
    // must be h3
    if (curr === null) {
      curr = {
        name: "Main",
        id: "main",
        a: [],
      }
      toc.push(curr);
    }
    const ael = [txt, id];
    curr.a.push(ael);
    //console.log(`${tag}, id: ${id}, '${txt}'`);
  }
  //let res = JSON.stringify(toc, null, "  ");
  //console.log(res);
  return genTocHTML(toc);
}

function processFile(srcPath, dstPath) {
  console.log(`Processing ${srcPath} => ${dstPath}`);
  //const decoder = new TextDecoder("utf-8");
  //const markdown = decoder.decode(await Deno.readFile(srcPath));
  let markdown = Deno.readTextFileSync(srcPath);
  Marked.setOptions({
    gfm: true,
    tables: true,
    langPrefix: "",
    highlight: (code, lang) => {
      const a = ["dosini", "fish", "nohighlight", "csv", "org", "jade", "textile"];
      const langSupported = lang && a.indexOf(lang) == -1;
      if (!langSupported) {
        return hljs.highlightAuto(code).value;
      }

      let opts = {
        language: lang,
        ignoreIllegals: true,
      }
      return hljs.highlight(code, opts).value;
    },
  });
  markdown = cleanupMarkdown(markdown);
  const markup = Marked.parse(markdown);
  //console.log(markup);
  let tocHTML = buildToc(markup.content);
  let startHTML = `
  <div id="start"></div>
  <div id="wrapped-content"></div>
`;
  let s = tocHTML + startHTML + `<div id="content">` + "\n" + markup.content + "\n" + `</div>`;
  s = genHTML(s, srcPath, markup.meta);
  Deno.writeTextFileSync(dstPath, s)
}

function processFiles() {
  //const files = ["go", "python", "bash", "101"];
  if (false) {
    genIndexHTML(files);
    return;
  }

  clean();

  const otherFiles = ["python3"];
  const allFiles = [];
  files.splice(8);
  for (let file of files) {
    let path = join("devhints", file + ".md");
    allFiles.push(path);
  }
  for (let file of otherFiles) {
    let path = join("other", file + ".md");
    allFiles.push(path);
  }

  for (let path of allFiles) {
    const src = path;
    const dst = basename(src).replace(".md", ".html");
    processFile(src, dst)
  }
  genIndexHTML(allFiles);
}

function cleanupMarkdown(s) {
  // remove lines like: {: data-line="1"}
  const reg = /{:.*}/g;
  s = s.replace(reg, "");
  s = s.replace("{% raw %}", "")
  s = s.replace("{% endraw %}", "");
  let prev = s;
  while (prev !== s) {
    prev = s;
    s = s.replace("\n\n", "\n");
  }
  return s;
}

function testCleanup() {
  const s = `## Objects
{: .-three-column}

### Example
{: .-prime}
`
  console.log(s);
  console.log("===>");
  const s2 = cleanupMarkdown(s);
  console.log(s2);
}

function clean() {
  for (const dirEntry of Deno.readDirSync('devhints')) {
    if (dirEntry.name.endsWith(".html")) {
      Deno.removeSync(dirEntry.name);
    }
  }
}

function listDevhints() {
  let a  = [];
  for (const dirEntry of Deno.readDirSync('devhints')) {
    if (dirEntry.isDirectory) {
      continue;
    }
    let name = dirEntry.name;
    if (!name.endsWith(".md")) {
      continue;
    }
    name = name.replace(".md", "");
    a.push(name);
  }
  const s = JSON.stringify(a, null, "  ");
  console.log(s);
}

//testCleanup();
processFiles();
//listDevhints();
