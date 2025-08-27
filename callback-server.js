const https = require('https');
const fs = require('fs');
const path = require('path');
const { parse } = require('url');

const httpsOptions = {
  key: fs.readFileSync(path.join(__dirname, 'certs', 'key.pem')),
  cert: fs.readFileSync(path.join(__dirname, 'certs', 'cert.pem')),
};

const server = https.createServer(httpsOptions, (req, res) => {
  const parsedUrl = parse(req.url, true);
  
  if (parsedUrl.pathname === '/callback') {
    const redirectUrl = `http://localhost:3000${req.url}`;
    res.writeHead(302, { 'Location': redirectUrl });
    res.end();
  } else {
    const redirectUrl = `http://localhost:3000${req.url}`;
    res.writeHead(302, { 'Location': redirectUrl });
    res.end();
  }
});

server.listen(3001, (err) => {
  if (err) throw err;
  console.log('🔐 Callback HTTPS server ready on https://localhost:3001');
});