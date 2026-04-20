// Task 4.1 — FR-301: BFF entry point — starts HTTP server
import { validateConfig, config } from './config';

validateConfig();

import app from './app';

const server = app.listen(config.port, () => {
  process.stdout.write(`FenixCRM BFF running on port ${config.port} [${config.nodeEnv}] → ${config.backendUrl}\n`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  server.close(() => {
    process.stdout.write('BFF server closed\n');
    process.exit(0);
  });
});

export default server;
