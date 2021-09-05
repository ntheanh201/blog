import { Marked } from "https://deno.land/x/markdown/mod.ts";
import hljs from "https://jspm.dev/highlight.js@11.0.1";
//import { default as hljs } from "https://cdn.skypack.dev/highlight.js";
// import { default as hlJavaScript } from "https://cdn.skypack.dev/highlight.js/lib/languages/javascript";
import { join } from "https://deno.land/std@0.106.0/path/mod.ts";
import { DOMParser, Element } from "https://deno.land/x/deno_dom/deno-dom-wasm.ts";

//console.log(hljs.listLanguages());
const lng = hljs.getLanguage("javascript");
//console.log(lng);
hljs.registerLanguage(lng.name, lng.rawDefinition);

// run as:
// deno run --allow-read --allow-write .\process.js

//console.log("highlight:");
//console.log(highlight);

// hljs.registerLanguage("javascript", hlJavaScript);

function genHTML(innerHTML, meta) {
  let name = meta.title;
  return `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8" />
    <title>${name} cheatsheet</title>
    <link href="s/main.css" rel="stylesheet" />
    <script src="s/main.js"></script>
</head>

<body onload="start()">
    <div class="breadcrumbs"><a href="/">Home</a> / <a href="index.html">cheatsheets</a> / ${name} cheatsheet</div>
    <!--
    <div class="edit">
        <a href="https://github.com/kjk/blog/blob/master/www/cheatsheets/python3.md" target="_blank">edit</a>
    </div>
    -->
    ${innerHTML}
</body>  
</html>`
}

function genIndexHTML(files) {
  let innerHTML = "";
  for (let file of files) {
    innerHTML += `<div><a href="${file}.html">${file}</a></div>`;
  }
  const s = `<!DOCTYPE html>
<html>

<head>
    <meta charset="utf-8" />
    <title>cheatsheets</title>
    <link href="s/main.css" rel="stylesheet" />
</head>

<div class="breadcrumbs"><a href="/">Home</a> / cheatsheets</div>

<div class="cslist">
  ${innerHTML}
</div>
</body>
</html>
`;
  const path = join("gen", "index.html");
  Deno.writeTextFileSync(path, s)
}

function genTocHTML(toc) {
  let html = `<div class="toc">`;
  for (let e of toc) {
    let s = `\n<b>${e.name}</b>: `;
    // TODO: handle a case where e.a is empty
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
      if (!lang) {
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
  let s = tocHTML + startHTML + `<div id="content">` + markup.content + `</div>`;
  s = genHTML(s, markup.meta);
  Deno.writeTextFileSync(dstPath, s)
}

function processFiles() {
  //const files = ["go", "python", "bash", "101"];
  const files = [
    "101",
    "absinthe",
    "activeadmin",
    "adb",
    "analytics.js",
    "analytics",
    "angularjs",
    "animated_gif",
    "ansi",
    "ansible-examples",
    "ansible-guide",
    "ansible-modules",
    "ansible-roles",
    "ansible",
    "appcache",
    "applescript",
    "applinks",
    "arel",
    "atom",
    "awesome-redux",
    "awscli",
    "backbone",
    "bash",
    "blessed",
    "bluebird",
    "bolt",
    "bookshelf",
    "bootstrap",
    "browser-sync",
    "browserify",
    "bulma",
    "bundler",
    "camp",
    "canvas",
    "capybara",
    "cask-index",
    "chai",
    "cheatsheet-styles",
    "chef",
    "chunky_png",
    "cidr",
    "circle",
    "co",
    "commander.js",
    "command_line",
    "composer",
    "cordova",
    "cron",
    "csharp7",
    "css-antialias",
    "css-flexbox",
    "css-grid",
    "css-system-font-stack",
    "css-tricks",
    "css",
    "cssnext",
    "curl",
    "c_preprocessor",
    "datetime",
    "deis",
    "deku",
    "deku@1",
    "devise",
    "divshot",
    "do",
    "docker-compose",
    "docker",
    "dockerfile",
    "dom-range",
    "dom-selection",
    "editorconfig",
    "elixir-metaprogramming",
    "elixir",
    "emacs",
    "ember",
    "emmet",
    "enzyme",
    "enzyme@2",
    "es6",
    "ets",
    "expectjs",
    "express",
    "exunit",
    "factory_bot",
    "fastify",
    "ffaker",
    "ffmpeg",
    "figlet",
    "find",
    "firebase",
    "firefox",
    "fish-shell",
    "flashlight",
    "flow",
    "flux",
    "flynn",
    "freenode",
    "frequency-separation-retouching",
    "gh-pages",
    "git-branch",
    "git-extras",
    "git-log-format",
    "git-log",
    "git-revisions",
    "git-tricks",
    "gnupg",
    "go",
    "goby",
    "google-webfonts",
    "google_analytics",
    "graphql",
    "gremlins",
    "gulp",
    "haml",
    "handlebars.js",
    "harvey.js",
    "heroku",
    "hledger",
    "homebrew",
    "html-email",
    "html-input",
    "html-meta",
    "html-microformats",
    "html-share",
    "html",
    "http-status",
    "httpie",
    "ie",
    "ie_bugs",
    "imagemagick",
    "immutable.js",
    "index",
    "index@2016",
    "inkscape",
    "ios-provision",
    "jade",
    "jasmine",
    "jekyll-github",
    "jekyll",
    "jest",
    "jquery-cdn",
    "jquery",
    "js-appcache",
    "js-array",
    "js-date",
    "js-fetch",
    "js-lazy",
    "js-model",
    "js-speech",
    "jscoverage",
    "jsdoc",
    "jshint",
    "knex",
    "koa",
    "kotlin",
    "kramdown",
    "layout-thrashing",
    "ledger-csv",
    "ledger-examples",
    "ledger-format",
    "ledger-periods",
    "ledger-query",
    "ledger",
    "less",
    "licenses",
    "linux",
    "lodash",
    "lua",
    "machinist",
    "macos-mouse-acceleration",
    "make-assets",
    "makefile",
    "man",
    "markdown",
    "meow",
    "meta-tags",
    "middleman",
    "minimist",
    "minitest",
    "mixpanel",
    "mobx",
    "mocha-blanket",
    "mocha-html",
    "mocha-tdd",
    "mocha",
    "modella",
    "modernizr",
    "moment",
    "mysql",
    "ncftp",
    "nock",
    "nocode",
    "nodejs-assert",
    "nodejs-fs",
    "nodejs-path",
    "nodejs-process",
    "nodejs-stream",
    "nodejs",
    "nopt",
    "npm",
    "org-mode",
    "osx",
    "package-json",
    "package",
    "pacman",
    "parsimmon",
    "parsley",
    "pass",
    "passenger",
    "perl-pie",
    "ph-food-delivery",
    "phoenix-conn",
    "phoenix-ecto",
    "phoenix-ecto@1.2",
    "phoenix-ecto@1.3",
    "phoenix-migrations",
    "phoenix-routing",
    "phoenix",
    "phoenix@1.2",
    "plantuml",
    "pm2",
    "polyfill.io",
    "postgresql-json",
    "postgresql",
    "premailer",
    "projectionist",
    "promise",
    "pry",
    "psdrb",
    "pug",
    "python",
    "qjs",
    "qunit",
    "rack-test",
    "ractive",
    "rails-controllers",
    "rails-forms",
    "rails-helpers",
    "rails-i18n",
    "rails-migrations",
    "rails-models",
    "rails-plugins",
    "rails-routes",
    "rails-tricks",
    "rails",
    "rake",
    "rbenv",
    "rdoc",
    "react-router",
    "react",
    "react@0.14",
    "README",
    "redux",
    "regexp",
    "rename",
    "resolutions",
    "rest-api",
    "riot",
    "rollup",
    "ronn",
    "rspec-rails",
    "rspec",
    "rst",
    "rsync",
    "rtorrent",
    "ruby",
    "ruby21",
    "rubygems",
    "sass",
    "saucelabs",
    "scp",
    "screen",
    "sed",
    "semver",
    "sequel",
    "sequelize",
    "sh-pipes",
    "sh",
    "shelljs",
    "siege",
    "simple_form",
    "sinon-chai",
    "sinon",
    "sketch",
    "slim",
    "social-images",
    "spacemacs",
    "spine",
    "spreadsheet",
    "sql-join",
    "stencil",
    "stimulus-reflex",
    "strftime",
    "stylus",
    "sublime-text",
    "superagent",
    "tabular",
    "tape",
    "textile",
    "tig",
    "tmux",
    "tomdoc",
    "top",
    "travis",
    "typescript",
    "ubuntu",
    "umdjs",
    "underscore-string",
    "unicode",
    "vagrant",
    "vagrantfile",
    "vainglory",
    "vim-diff",
    "vim-digraphs",
    "vim-easyalign",
    "vim-help",
    "vim-rails",
    "vim-unite",
    "vim",
    "vimscript-functions",
    "vimscript-snippets",
    "vimscript",
    "virtual-dom",
    "vows",
    "vscode",
    "vue",
    "vue@1.0.28",
    "watchexec",
    "watchman",
    "web-workers",
    "webpack",
    "weechat",
    "weinre",
    "xpath",
    "yaml",
    "yargs",
    "yarn",
    "znc",
    "zombie",
    "zsh"
  ];

  for (let file of files) {
    const src = join("devhints", file + ".md")
    const dst = join("gen", file + ".html");
    processFile(src, dst)
  }
  genIndexHTML(files);
}

function cleanupMarkdown(s) {
  // remove lines like: {: data-line="1"}
  const reg = /{:.*}/g;
  s = s.replace(reg, "");
  s = s.replace("\n\n", "\n");
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
//processFiles();
listDevhints();
