import http from "node:http";
import fs from "node:fs";
import path from "node:path";
import net from "node:net";
import { fileURLToPath } from "node:url";
import server from "./dist/server/server.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const clientDir = path.join(__dirname, "dist", "client");

const MIME_TYPES = {
  ".js": "application/javascript",
  ".mjs": "application/javascript",
  ".css": "text/css",
  ".html": "text/html",
  ".json": "application/json",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".jpeg": "image/jpeg",
  ".gif": "image/gif",
  ".svg": "image/svg+xml",
  ".ico": "image/x-icon",
  ".woff": "font/woff",
  ".woff2": "font/woff2",
  ".ttf": "font/ttf",
  ".eot": "application/vnd.ms-fontobject",
  ".webp": "image/webp",
  ".map": "application/json",
};

const port = parseInt(process.env.PORT || "3000", 10);
const host = process.env.HOST || "0.0.0.0";
const apiUrl = process.env.API_URL || "http://localhost:8080";

const httpServer = http.createServer((req, res) => {
  const url = new URL(req.url, `http://${req.headers.host || "localhost"}`);

  if (url.pathname.startsWith("/api/") || url.pathname.startsWith("/api/ws/")) {
    proxyRequest(req, res, url);
    return;
  }

  if (isStaticFileRequest(url.pathname)) {
    serveStaticFile(url.pathname, req, res);
    return;
  }

  handleSSRRequest(req, res, url);
});

function isStaticFileRequest(pathname) {
  if (pathname.startsWith("/assets/")) return true;
  if (pathname.match(/\.\w+$/)) return true;
  return false;
}

function serveStaticFile(pathname, req, res) {
  const filePath = path.join(clientDir, pathname);

  if (!filePath.startsWith(clientDir)) {
    res.statusCode = 403;
    res.end("Forbidden");
    return;
  }

  fs.readFile(filePath, (err, data) => {
    if (err) {
      res.statusCode = 404;
      res.end("Not Found");
      return;
    }

    const ext = path.extname(filePath).toLowerCase();
    const contentType = MIME_TYPES[ext] || "application/octet-stream";
    res.statusCode = 200;
    res.setHeader("Content-Type", contentType);
    res.setHeader("Cache-Control", "public, max-age=31536000, immutable");
    res.end(data);
  });
}

function handleSSRRequest(req, res, url) {
  const headers = new Headers();
  for (const [key, value] of Object.entries(req.headers)) {
    if (Array.isArray(value)) {
      for (const v of value) headers.append(key, v);
    } else if (value) {
      headers.set(key, value);
    }
  }

  if (req.method !== "GET" && req.method !== "HEAD") {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      const body = chunks.length > 0 ? Buffer.concat(chunks) : null;
      forwardToSSR(req, res, url, headers, body);
    });
  } else {
    forwardToSSR(req, res, url, headers, null);
  }
}

async function forwardToSSR(req, res, url, headers, body) {
  const webReq = new Request(url.toString(), {
    method: req.method,
    headers,
    body,
  });

  try {
    const webRes = await server.fetch(webReq);
    res.statusCode = webRes.status;
    for (const [key, value] of webRes.headers.entries()) {
      res.setHeader(key, value);
    }
    if (webRes.body) {
      const reader = webRes.body.getReader();
      const writer = (chunk) => {
        if (chunk.done) {
          res.end();
          return;
        }
        res.write(chunk.value);
        return reader.read().then(writer);
      };
      await reader.read().then(writer);
    } else {
      res.end();
    }
  } catch (err) {
    console.error("Server error:", err);
    res.statusCode = 500;
    res.end("Internal Server Error");
  }
}

async function proxyRequest(req, res, url) {
  const proxyUrl = new URL(url.pathname + url.search, apiUrl);

  const headers = { ...req.headers, host: new URL(apiUrl).host };
  delete headers["connection"];

  const chunks = [];
  if (req.method !== "GET" && req.method !== "HEAD") {
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      const body = chunks.length > 0 ? Buffer.concat(chunks) : undefined;
      doProxy(req, res, proxyUrl, headers, body);
    });
  } else {
    doProxy(req, res, proxyUrl, headers, undefined);
  }
}

function doProxy(req, res, proxyUrl, headers, body) {
  const options = {
    hostname: proxyUrl.hostname,
    port: proxyUrl.port,
    path: proxyUrl.pathname + proxyUrl.search,
    method: req.method,
    headers,
  };

  const proxyReq = http.request(options, (proxyRes) => {
    res.writeHead(proxyRes.statusCode, proxyRes.headers);
    proxyRes.pipe(res, { end: true });
  });

  proxyReq.on("error", (err) => {
    console.error("Proxy error:", err);
    res.statusCode = 502;
    res.end("Bad Gateway");
  });

  if (body) proxyReq.write(body);
  proxyReq.end();
}

httpServer.listen(port, host, () => {
  console.log(`Frontend server listening on http://${host}:${port}`);
  console.log(`Proxying /api/* to ${apiUrl}`);
});

httpServer.on("upgrade", (req, socket, head) => {
  if (req.url && req.url.startsWith("/api/ws/")) {
    const target = new URL(apiUrl);
    const options = {
      hostname: target.hostname,
      port: target.port,
      path: req.url,
      method: req.method,
      headers: {
        ...req.headers,
        host: target.host,
        connection: "upgrade",
        upgrade: "websocket",
      },
    };

    const proxyReq = http.request(options);
    proxyReq.on("upgrade", (proxyRes, proxySocket) => {
      socket.write(
        "HTTP/1.1 101 Switching Protocols\r\n" +
        Object.entries(proxyRes.headers)
          .map(([k, v]) => `${k}: ${v}`)
          .join("\r\n") +
        "\r\n\r\n"
      );
      proxySocket.pipe(socket, { end: true });
      socket.pipe(proxySocket, { end: true });
    });
    proxyReq.on("error", (err) => {
      console.error("WS proxy error:", err);
      socket.end();
    });
    proxyReq.end();
  } else {
    socket.destroy();
  }
});
