#!/usr/bin/env node
/**
 * 本地代理服务器 — 解决 file:// 协议的跨域问题
 * 
 * 用法: node proxy-server.mjs
 * 页面: http://localhost:3000/
 * 所有 /api/* 请求自动转发到 localhost:8080
 */
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const PORT = 3000;
const BACKEND = 'http://localhost:8080';
const __dirname = path.dirname(fileURLToPath(import.meta.url));

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
};

function serveFile(res, filePath) {
  const ext = path.extname(filePath);
  const mime = MIME[ext] || 'application/octet-stream';
  try {
    const content = fs.readFileSync(filePath);
    res.writeHead(200, { 'Content-Type': mime, 'Access-Control-Allow-Origin': '*' });
    res.end(content);
  } catch {
    res.writeHead(404);
    res.end('Not Found');
  }
}

function proxyAPI(req, res) {
  const options = new URL(req.url, BACKEND);
  const proxyReq = http.request(
    {
      hostname: options.hostname,
      port: options.port,
      path: options.pathname + options.search,
      method: req.method,
      headers: { ...req.headers, host: options.host },
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode, proxyRes.headers);
      proxyRes.pipe(res);
    }
  );
  proxyReq.on('error', (err) => {
    res.writeHead(502, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'backend_unreachable', detail: err.message }));
  });
  req.pipe(proxyReq);
}

const server = http.createServer((req, res) => {
  console.log(`${req.method} ${req.url}`);

  // API 请求 → 转发后端
  if (req.url.startsWith('/api/')) {
    return proxyAPI(req, res);
  }

  // 根路径 → v4-web.html
  if (req.url === '/' || req.url === '/index.html') {
    return serveFile(res, path.join(__dirname, 'v4-web.html'));
  }

  // 其他静态文件
  const safePath = path.normalize(req.url).replace(/^\/+/, '');
  serveFile(res, path.join(__dirname, safePath));
});

server.listen(PORT, () => {
  console.log(`
╔══════════════════════════════════════════════╗
║  本地代理服务器已启动                          ║
║                                              ║
║  前端: http://localhost:${PORT}                    ║
║  API:  转发至 ${BACKEND}        ║
║                                              ║
║  按 Ctrl+C 停止                               ║
╚══════════════════════════════════════════════╝
`);
});
