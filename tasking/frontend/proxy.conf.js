const PROXY_CONFIG = [
  {
    context: ['/api/orders', '/api/feasibility'],
    target: 'http://127.0.0.1:8080',
    secure: false,
  },
  {
    context: ['/api'],
    target: 'http://127.0.0.1:8081',
    secure: false,
  },
];

module.exports = PROXY_CONFIG;
