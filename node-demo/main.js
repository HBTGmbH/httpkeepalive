const http = require('http');
const DEFAULT_MIN = 10;
const DEFAULT_MAX = 500;
const ENABLE_DRAIN = process.env.ENABLE_DRAIN === 'true' || false;
const DRAIN_TIMEOUT = Number(process.env.DRAIN_TIMEOUT) || 5000;
const KEEP_ALIVE_TIMEOUT = Number(process.env.KEEP_ALIVE_TIMEOUT) || 5000;
let draining = false;

console.log("KEEP_ALIVE_TIMEOUT:", KEEP_ALIVE_TIMEOUT);
console.log("ENABLE_DRAIN:", ENABLE_DRAIN);
console.log("DRAIN_TIMEOUT:", DRAIN_TIMEOUT);

function respond(res, status, msg) {
  res.statusCode = status
  res.setHeader('Content-Type', 'text/plain')
  if (ENABLE_DRAIN && draining) {
    res.setHeader('Connection', 'close')
  }
  return res.end(msg);
}

function getConnectionsAsync(server) {
  return new Promise((resolve, reject) => {
    server.getConnections((err, count) => {
      if (err) {
        reject(err);
      } else {
        resolve(count);
      }
    });
  });
}

function delay(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

const server = http.createServer({ keepAliveTimeout: KEEP_ALIVE_TIMEOUT },
  (req, res) => {
    const { pathname, searchParams } = new URL(req.url, 'http://localhost');
    if (pathname === '/ready') {
      return respond(res, 200, '')
    } else if (pathname === '/sleep') {
      let min = Math.floor(Number(searchParams.get('min'))) || DEFAULT_MIN;
      let max = Math.floor(Number(searchParams.get('max'))) || DEFAULT_MAX;
      const range = max - min;
      const delay = range > 0 ? min + Math.floor(Math.random() * (range + 1)) : min;
      setTimeout(() => {
        return respond(res, 200, `Slept for ${String(delay)} ms.\n`);
      }, delay);
    } else if (pathname === '/echo') {
      return respond(res, 200, JSON.stringify({
        method: req.method,
        url: req.url,
        headers: req.headers
      }, null, 2) + '\n');
    } else {
      return respond(res, 404, '')
    }
  }
);

async function drainConnections() {
  draining = true;
  try {
    while (true) {
      let count = await getConnectionsAsync(server);
      if (count <= 0) {
        console.log(`All connections gone. Closing server`);
        server.close(() => { console.log('Server closed'); })
        return;
      } else {
        console.log(`There are still ${count} connections. Keep draining...`);
        await delay(500);
      }
    }
  } catch (err) {
    console.error('Failed to get connections:', err);
  }
}

function gracefulShutdown(sig) {
  if (!ENABLE_DRAIN || DRAIN_TIMEOUT <= 0) {
    console.log(`${sig} signal received. Closing server.`);
    server.close(() => { console.log('Server closed'); })
  } else {
    console.log(`${sig} signal received. Draining connections...`);
    drainConnections();
  }
}

process.on('SIGTERM', () => {
  gracefulShutdown('SIGTERM')
});
process.on('SIGINT', () => {
  gracefulShutdown('SIGINT')
});
server.listen(3000);
console.log('Server listening on port 3000');

