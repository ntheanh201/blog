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


function getRandomInt(max) {
    return Math.floor(Math.random() * max);
}

function focusSearch() {
    const el = document.getElementById("cs-search-input");
    el.focus();
}
