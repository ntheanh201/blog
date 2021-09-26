import { getAssetFromKV, mapRequestToAsset } from '@cloudflare/kv-asset-handler'
import redirects from './redirects.js';

/**
 * The DEBUG flag will do two things that help during development:
 * 1. we will skip caching on the edge, which makes it easier to
 *    debug.
 * 2. we will return an error message on exception in your Response rather
 *    than the default 404.html page.
 */
const DEBUG = false

addEventListener('fetch', fetchEventListener);

function fetchEventListener(event) {
  try {
    event.respondWith(handleEvent(event))
  } catch (e) {
    if (DEBUG) {
      return event.respondWith(
        new Response(e.message || e.toString(), {
          status: 500,
        }),
      )
    }
    event.respondWith(new Response('Internal Error', { status: 500 }))
  }
}

function setHeaders(response) {
  response.headers.set("X-XSS-Protection", "1; mode=block");
  response.headers.set("X-Content-Type-Options", "nosniff");
  response.headers.set("X-Frame-Options", "DENY");
  response.headers.set("Referrer-Policy", "unsafe-url");
  response.headers.set("Feature-Policy", "none");
}

function getRedirectInfo(path) {
  const a = redirects[path];
  if (a) {
    return a;
  }

  // /article/:id/${title} => /article/:id/${title}.html
  if (path.endsWith(".html") || !path.startsWith("/article/")) {
    return null;
  }

  // "/article/abbbcb44f6fd4ba5bdb04b3970180958/tesla-facts"
  // =>
  // ['', 'article', 'abbbcb44f6fd4ba5bdb04b3970180958', 'tesla-facts']
  const parts = path.split("/");
  if (parts.length !== 4) {
    return null;
  }
  const newURL =  path + ".html";
  console.log("getRedirectInfo: ", path, "=>", newURL);
  return [newURL, 200];
}

async function maybeRedirect(event) {
  const url = new URL(event.request.url);
  const a = getRedirectInfo(url.pathname);
  if (!a) {
    return null;
  }
  const newURL = a[0];
  const code = a[1];
  if (code === 302) {
    const response = new Response("", { status: 302 });
    response.headers.set("Location", newURL);
    return response;
  }
  // assuming this is 200
  try {
    function mapReqToAssetFunc(req) {
      const reqURL = `${new URL(req.url).origin}${newURL}`;
      const ret = new Request(reqURL, req);
      return ret;
    }
    let opts = {
      mapRequestToAsset: mapReqToAssetFunc,
    };
    const page = await getAssetFromKV(event, opts);
    // allow headers to be altered
    const response = new Response(page.body, page);
    setHeaders(response);
    return response;
  } catch (e) {
    // do nothing
  }
  return null;
}

async function handleEvent(event) {
  //console.log("url:", event.request.url);
  const redirectRsp = await maybeRedirect(event);
  if (redirectRsp != null) {
    logdna(event, 200);
    return redirectRsp;
  }

  let options = {}

  /**
   * You can add custom logic to how we fetch your assets
   * by configuring the function `mapRequestToAsset`
   */

  try {
    if (DEBUG) {
      // customize caching
      options.cacheControl = {
        bypassCache: true,
      };
    }
    const page = await getAssetFromKV(event, options);
    // allow headers to be altered
    const response = new Response(page.body, page);
    setHeaders(response);
    logdna(event, 200);
    return response;
  } catch (e) {
    // if an error is thrown try to serve the asset at 404.html
    if (!DEBUG) {
      try {
        let notFoundResponse = await getAssetFromKV(event, {
          mapRequestToAsset: req => new Request(`${new URL(req.url).origin}/404.html`, req),
        })

        logdna(event, 400);
        return new Response(notFoundResponse.body, { ...notFoundResponse, status: 404 })
      } catch (e) {}
    }

    logdna(event, 500);

    return new Response(e.message || e.toString(), { status: 500 })
  }
}

// we don't care to log access to files that end with this extension
const extsToFilter = [
  ".png",
  ".jpg",
  ".css",
  "/ping",
  ".txt",
  ".xml"
];

const containsToFilter = [
  "/app/crashsubmit",
  "wp-login.php",
  "xmlrpc.php",
  "/favicon.ico",
  "/forum_sumatra",
  "/software/sumatrapdf"
];

function shouldSkipLoggingOf(request, statusCode) {
  const uri = request.url;
  for (let ext of extsToFilter) {
    if (uri.endsWith(ext)) {
      return true;
    }
  }
  for (let s of containsToFilter) {
    if (uri.includes(s)) {
      return true;
    }
  }
  return false;
}

function logdna(event, statusCode) {
  const request = event.request;
  //console.log("logdna:", request.url);
  // hostname cannot have dots in it so can't do blog.kowalczyk.info
  const hostname = "blog";
  const app = "blog";
  const apiKey = LOGDNA_INGESTION_KEY;
  if (!apiKey) {
    return;
  }
  if (shouldSkipLoggingOf(request, statusCode)) {
    return;
  }
  const line = {
    "line": request.url,
    "app": app,
    "timestamp": Date.now(),
  };
  if (statusCode >= 300) {
    line["level"] = "ERROR";
  }
  const meta = {
    "code": statusCode,
  };
  const referer = request.headers.get("referer");
  if (referer) {
    meta["referer"] = referer;
  }
  const ua = request.headers.get("user-agent");
  if (ua) {
    meta["ua"] = ua;
  }
  const ip = request.headers.get("cf-connecting-ip");
  if (ip) {
    meta["ip"] = ip;
  }
  if (request.cf) {
    if (request.cf.country) {
      meta["country"] = request.cf.country;
    }
    if (request.cf.city) {
      meta["city"] = request.cf.city;
    }
  }

  line["meta"] = meta;
  const payload = {
    "lines": [line],
  };
  const opts = {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json;charset=utf-8'
    },
    body: JSON.stringify(payload),
  };
  let uri = `https://logs.logdna.com/logs/ingest?hostname=${hostname}&apikey=${apiKey}`;
  // set it here so that we can wa
  try {
    let rsp = fetch(uri, opts);
    event.waitUntil(rsp);
  } catch {
    // no-op
  }
}
