const http = require('node:http');

// Create a local server to receive data from
const server = http.createServer((req, res) => {

    console.log("Incoming request", req.headers)

    const username = req.headers["x-forwarded-preferred-username"]

    res.writeHead(200, { 'Content-Type': 'application/json' });

    res.end(JSON.stringify({
        message: `Hello ${username || "Unkown Somebody"}!`
    }));
});

server.listen(8000);
