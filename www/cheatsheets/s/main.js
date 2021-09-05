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

// for nested headers we want the name be "File manipulation / Reading"
// instead of just "Reading".
// we build those once on h3 / h4 elements and store as "data-name" attribute
function buildHeaderFullNames() {
    let currH2 = "";
    let currH3 = "";
    const parent = elById("#content");
    for (const el of parent.children) {
        const tag = el.localName;
        if (tag === "h2") {
            currH2 = el.textContent;
            currH3 = "";
            continue;
        }
        if (tag === "h3") {
            if (currH2 === "") {
                currH3 = el.textContent;
            } else {
                currH3 = currH2 + " / " + el.textContent;
            }
            el.setAttribute("data-name", currH3);
            continue;
        }

        if (tag === "h4") {
            const currH4 = currH3 + " / " + el.textContent;
            el.setAttribute("data-name", currH4);
            continue;
        }
    }
}

// for every h2 in #content, wraps it and it's siblings (until next h2)
// inside div and appends that div to #start
function groupHeaderElements() {
    const parent = elById("#content");
    const groups = [];
    let curr = [];
    for (const el of parent.children) {
        if (el.localName === "h2" || el.localName === "h3") {
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

function bringToFrontDiv(hdrId) {
    // remove "first" class from the element that currently has it
    let els = document.getElementsByClassName("first");
    for (let el of els) {
        el.classList.remove("first");
    }

    const divId = hdrId + "-wrap";
    let el = elById(divId);
    el.classList.add("first");
    const startEl = elById("#start");
    el.remove();
    startEl.after(el);

    el = elById(hdrId)
    const fullName = el.getAttribute("data-name");
    if (fullName && fullName !== el.textContent) {
        el.textContent = fullName;
    }
    el.classList.add("flash");
    //updateLocationHash(divId);
}

function bringToFront(target) {
    const hdrId = target.getAttribute("href");
    bringToFrontDiv(hdrId);
}

function onClick(ev) {
    ev.preventDefault();
    bringToFront(ev.target);
    window.scrollTo(0, 0);
}

function filterList(s) {
    const els = document.getElementsByClassName("cslist-item");
    for (const el of els) {
        const v = el.getElementsByTagName("a");
        if (len(v) != 1) {
            continue;
        }
        const a = v[0];
        const txt = a.textContent;
        let disp = txt.includes(s) ? "block" : "none";
        if (!s || s === "") {
            disp = "block";
        }
        el.style.display = disp;
    }
    return s;
}

function hookClick() {
    const els = document.getElementsByTagName("a");
    const n = els.length;
    for (let i = 0; i < n; i++) {
        const el = els.item(i);
        const href = el.getAttribute("href");
        if (!href) {
            continue;
        }
        if (href[0] == "#") {
            el.onclick = onClick;
        } else if (href.startsWith("http")) {
            // make all external links open in new tab
            el.setAttribute("target", "_blank");
        }
    }
}
async function start() {
    buildHeaderFullNames(); // must call before groupHeaderElements()
    groupHeaderElements();
    hookClick();
    const el = document.getElementById("intro");
    if (el) {
        bringToFrontDiv("intro");
    }
}

async function startIndex() {
    const el = document.getElementById("search-input");
    el.focus();
}