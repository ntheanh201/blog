function byId(id) {
  return document.getElementById(id);
}

var serverBase = "https://onlinetool.io";

var monacoURLBase =
  "https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.25.2/min/vs";

require.config({
  paths: { vs: monacoURLBase }
});

window.MonacoEnvironment = { getWorkerUrl: () => proxy };

let proxy = URL.createObjectURL(
  new Blob(
    [
      `
	self.MonacoEnvironment = {
		baseUrl: 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.25.2/min/'
	};
	importScripts('https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.25.2/min/vs/base/worker/workerMain.js');
`
    ],
    { type: "text/javascript" }
  )
);

var editorGo;
var editorXml;

function clearMsgOrErr() {
  var el = byId("msgOrErr");
  el.innerHTML = "&nbsp;";
}

function clearMsgOrErrDelayed() {
  setInterval(clearMsgOrErr, 5000);
}

function showErr(err) {
  var el = byId("msgOrErr");
  el.textContent = "Error: '" + err + "'";
  el.style.color = "red";
}

function showMsg(msg) {
  var el = byId("msgOrErr");
  el.textContent = msg;
  el.style.color = "gray";
}

function disableURLEntry() {
  var el = byId("gourl");
  el.disabled = true;
  el = byId("gobtn");
  el.disabled = true;
}

function eanbleURLEntry() {
  var el = byId("gourl");
  el.disabled = false;
  el.value = "";
  el = byId("gobtn");
  el.disabled = false;
}

function onConvertResult(xhr, response) {
  //console.log("onConvertResult:", response);
  if (response.error) {
    showErr(response.error);
    clearMsgOrErr();
    return;
  }
  // TODO: show how long it took
  showMsg("Received result from the server.");
  clearMsgOrErrDelayed();
  var model = editorGo.getModel();
  model.setValue(response.go || "");
  monaco.editor.setModelLanguage(model, "go");
}

function onConvertResultError(e, xhr, response) {
  //console.log("onConvertResultError:", e);
  var model = editorGo.getModel();
  model.setValue(response.error);
  monaco.editor.setModelLanguage(model, "text");
}

function xmlToGo(xml) {
  var data = { xml: xml };
  var opts = { responseType: "json" };
  showMsg("Sending data to the server...");
  qwest
    .post(serverBase + "/xmltogo/convert", data, opts)
    .then(onConvertResult)
    .catch(onConvertResultError);
}
// e: https://microsoft.github.io/monaco-editor/api/interfaces/monaco.editor.imodelcontentchangedevent.html
function xmlChanged(e) {
  var xmlModel = editorXml.getModel();
  // xmlModel: https://microsoft.github.io/monaco-editor/api/interfaces/monaco.editor.itextmodel.html
  var s = xmlModel.getValue();

  // This is a work-around. XML files pasted into the box are utf-8 and this
  // confuses zek. Not sure what happens if downloaded XML is marked as UTF-16
  // and actually is UTF-16. Will monaco magically convert it to utf-8?
  s = s.replace(/encoding="utf-16"/i, "");

  // console.log("xml changed:", s);
  xmlToGo(s);
}

function onXmlDlResult(xhr, response) {
  //console.log("onXmlDlResult:", response);
  eanbleURLEntry();
  if (response.error) {
    showErr(response.error);
    return;
  }
  var model = editorXml.getModel();
  model.setValue(response.xml);
  monaco.editor.setModelLanguage(model, "xml");
}

function onXmlDlError(e, xhr) {
  //console.log("onXmlDlError: e", e);
  eanbleURLEntry();
  showErr(e);
}

function goKeyUp(e) {
  if (e.keyCode === 13) {
    var el = byId("gobtn");
    el.click();
  }
}

function goClicked(e) {
  clearMsgOrErr();
  var el = byId("gourl");
  var uri = el.value.trim();
  //console.log("uri:", uri);
  if (uri == "") {
    return;
  }
  showMsg("Downloading XML...");
  disableURLEntry();
  var data = { url: uri };
  var opts = { responseType: "json" };
  qwest
    .post("/xmltogo/dlxml", data, opts)
    .then(onXmlDlResult)
    .catch(onXmlDlError);
}

function initEditor() {
  var el = byId("editor-xml");
  editorXml = monaco.editor.create(el, {
    value: "<xml>\n\t<body></body>\n</xml>\n",
    minimap: {
      enabled: false
    },
    language: "xml"
  });

  el = byId("editor-go");
  // options: https://microsoft.github.io/monaco-editor/api/interfaces/monaco.editor.ieditorconstructionoptions.html
  editorGo = monaco.editor.create(el, {
    value: "",
    readOnly: true,
    lineNumbers: "off",
    minimap: {
      enabled: false
    },
    language: "go"
  });

  var xmlModel = editorXml.getModel();
  xmlModel.onDidChangeContent(xmlChanged);
  // trigger generation on first load
  xmlChanged();
}

function init() {
  require(["vs/editor/editor.main"], initEditor);
}

document.addEventListener("DOMContentLoaded", init);
