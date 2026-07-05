#!/usr/bin/env node

const http = require("http");
const port = parseInt(process.env.PORT || "13714");
const host = process.env.HOST || "127.0.0.1";

const server = http.createServer(async (req, res) => {
  if (req.method !== "POST" || req.url !== "/render") {
    res.writeHead(404);
    res.end();
    return;
  }

  let body = "";
  req.on("data", (chunk) => (body += chunk));
  req.on("end", async () => {
    try {
      const page = JSON.parse(body);
      const { render } = await import("../../../public/build/assets/ssr/ssr.mjs");
      const result = await render(page);
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ head: result.head, body: result.body }));
    } catch (e) {
      res.writeHead(500, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: e.message }));
    }
  });
});

server.listen(port, host, () => {
  console.log(`Inertia SSR server running on http://${host}:${port}`);
});
