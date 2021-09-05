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
    window.scrollTo(0, 0);
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
        } else {
            // make all external links open in new tab
            el.setAttribute("target", "_blank");
        }
    }
}
async function start() {
    groupHeaderElements();
    hookClick();
    bringToFrontDiv("intro");
}
