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

addEventListener('fetch', event => {
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
})

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
  // /article/:id/* => /article/:id.html

  if (!path.startsWith("/article/")) {
    return null;
  }
  const parts = path.split("/");
  // console.log("getRedirectInfo: path:", parts);
  if (parts.length !== 4) {
    return null;
  }
  const id = parts[2];
  const newURL =  "/article/" + id + ".html";
  //console.log("getRedirectInfo: newURL:", newURL);
  return [newURL, 200];
}

async function maybeRedirect(event) {
  const url = new URL(event.request.url);
  //console.log(`maybeRedirect: url.pathname: ${url.pathname}`);
  const a = getRedirectInfo(url.pathname);
  if (!a) {
    return null;
  }
  const newURL = a[0];
  const code = a[1];
  //console.log(`maybeRedirect, newURL: ${newURL}, code: ${code}`);
  if (code === 302) {
    const response = new Response("", { status: 302 });
    response.headers.set("Location", newURL);
    return response;
  }
  // assuming this is 200
  try {
    function mapReqToAssetFunc(req) {
      const reqURL = `${new URL(req.url).origin}${newURL}`;
      //console.log(`maybeRedirect: reqURL: ${reqURL}`);
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
  //console.log(`handleEvent: ${event.request.url}`);

  const redirectRsp = await maybeRedirect(event);
  if (redirectRsp != null) {
    return redirectRsp;
  }

  const url = new URL(event.request.url)

  let options = {}

  /**
   * You can add custom logic to how we fetch your assets
   * by configuring the function `mapRequestToAsset`
   */
  // options.mapRequestToAsset = handlePrefix(/^\/docs/)

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
    return response;
  } catch (e) {
    // if an error is thrown try to serve the asset at 404.html
    if (!DEBUG) {
      try {
        let notFoundResponse = await getAssetFromKV(event, {
          mapRequestToAsset: req => new Request(`${new URL(req.url).origin}/404.html`, req),
        })

        return new Response(notFoundResponse.body, { ...notFoundResponse, status: 404 })
      } catch (e) {}
    }

    return new Response(e.message || e.toString(), { status: 500 })
  }
}

/**
 * Here's one example of how to modify a request to
 * remove a specific prefix, in this case `/docs` from
 * the url. This can be useful if you are deploying to a
 * route on a zone, or if you only want your static content
 * to exist at a specific path.
 */
function handlePrefix(prefix) {
  return request => {
    // compute the default (e.g. / -> index.html)
    let defaultAssetKey = mapRequestToAsset(request)
    let url = new URL(defaultAssetKey.url)

    // strip the prefix from the path for lookup
    url.pathname = url.pathname.replace(prefix, '/')

    // inherit all other props from the default request
    return new Request(url.toString(), defaultAssetKey)
  }
}