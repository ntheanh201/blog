import { Marked } from "https://deno.land/x/markdown/mod.ts";
//import highlight from "https://jspm.dev/highlight.js@11.0.1";
import { default as highlight } from "https://cdn.skypack.dev/highlight.js";
import { join } from "https://deno.land/std@0.106.0/path/mod.ts";

// run as:
// deno run --allow-read --allow-write .\process.js

//console.log("highlight:");
//console.log(highlight);

async function processFile(srcPath, dstPath) {
  console.log(`Processing ${srcPath} => ${dstPath}`);
  //const decoder = new TextDecoder("utf-8");
  //const markdown = decoder.decode(await Deno.readFile(srcPath));
  const markdown = Deno.readTextFileSync(srcPath);
  Marked.setOptions({
    gfm: true,
    tables: true,
    highlight: (code, lang) => {
      let opts = {
        language: lang,
      }
      highlight.highlight(code, opts).value;
    },
  });
  const markup = Marked.parse(markdown);
  //console.log(markup);
  Deno.writeTextFileSync(dstPath, markup.content)
}

const files = ["101"];

for (let file of files) {
  const src = join("devhints", file + ".md")
  const dst = join("gen", file + ".html");
  processFile(src, dst)
}
//console.log(markup.content);
//console.log(JSON.stringify(markup.meta))
