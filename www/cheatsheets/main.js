function len(o) {
    if (o && o.length) {
        return o.length;
    }
    return 0;
}

function elById(id) {
    if (id[0] == "#") {
        id = id.substring(1);
    }
    return document.getElementById(id);
}

function updateLocationHash(divId) {
    let h = window.location.hash;
    if (len(h) == 0) {
        h = divId;
    } else {
        h = divId + ";" + h.substr(1);
    }
    window.location.hash = h;
}

function bringToFrontDiv(h2Id) {
    let els = document.getElementsByClassName("first");
    for (let el of els) {
        el.classList.remove("first");
    }

    const divId = h2Id + "-wrap";
    let el = elById(divId);
    el.classList.add("first");
    const startEl = elById("#start");
    el.remove();
    startEl.after(el);

    el = elById(h2Id)
    el.classList.add("flash");
    //updateLocationHash(divId);
}

function bringToFront(target) {
    const h2Id = target.getAttribute("href");
    bringToFrontDiv(h2Id);
}

function onClick(ev) {
    ev.preventDefault();
    bringToFront(ev.target);
}

function hookClick() {
    const els = document.getElementsByTagName("a");
    const n = els.length;
    for (let i = 0; i < n; i++) {
        const el = els.item(i);
        const href = el.getAttribute("href");
        if (!href || href[0] != "#") {
            continue;
        }
        el.onclick = onClick;
    }
}

// for every h2 in #content, wraps it and it's siblings (until next h2)
// inside div and appends that div to #start
function groupH2Elements() {
    const parent = elById("#content");
    const groups = [];
    let curr = [];
    for (const el of parent.children) {
        if (el.localName === "h2") {
            if (curr.length > 0) {
                groups.push(curr);
            }
            curr = [el];
        } else {
            curr.push(el);
        }
    }
    if (curr.length > 0) {
        groups.push(curr);
    }

    for (const group of groups) {

        const div = document.createElement("div");
        div.id = group[0].id + "-wrap";
        div.className = "dvwrap";

        for (const el of group) {
            div.appendChild(el);
        }

        const parent = elById("#wrapped-content");
        parent.appendChild(div);
    }
}

function highlightCode(lang) {
    const els = document.getElementsByTagName("code");
    const n = els.length;
    for (let i = 0; i < n; i++) {
        const el = els.item(i);
        let ignore = el.classList.contains("ignore");
        if (ignore) {
            continue;
        }
        el.className = lang;
        hljs.highlightElement(el);
    }
}

async function start(file, lang, mdconv) {
    const rsp = await fetch(file);
    const mdTxt = await rsp.text();

    if (mdconv == "markdown-it") {
        const opts = {
            html: true,
        };
        const md = window.markdownit("default", opts);
        md.use(window.markdownItAnchor, {
            permalinke: true,
        });
        md.use(window.markdownitAttrs);
        const html = md.render(mdTxt);
        const el = elById("#content");
        el.innerHTML = html;
    } else {
        // TODO: can remove
        const opts = {
            "tables": true,
        };
        const converter = new showdown.Converter(opts);
        const html = converter.makeHtml(mdTxt);
        const el = elById("#content");
        el.innerHTML = html;
    }

    groupH2Elements();
    highlightCode(lang);
    hookClick();
    if (mdconv == "markdown-it") {
        const elToc = elById("toc");
        const el2 = elToc.nextSibling;
        console.log(elToc);
        console.log(el2);
        const elStart = elById("start");
        console.log(elStart);
        elStart.appendChild(el2);
        
        const elTocWrap = elById("toc-wrap");
        elTocWrap.remove()
    }
}

async function startPython() {
    await start("python3.md", "python", "markdown-it");
    bringToFrontDiv("basic-script-template");
}

async function startGo() {
    await start("go.md", "go", "markdown-it");
    bringToFrontDiv("hello-world");
}
