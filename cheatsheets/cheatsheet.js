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

function getRandomInt(max) {
    return Math.floor(Math.random() * max);
}

function focusSearch() {
    const el = document.getElementById("cs-search-input");
    el.focus();
}
