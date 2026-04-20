// Task 4.1 — FR-301: BFF entry point — starts HTTP server
import { validateConfig, config } from './config';

validateConfig();

import app from './app';

const server = app.listen(config.port, () => {
  // eslint-disable-next-line no-console
  console.log(`FenixCRM BFF running on port ${config.port} [${config.nodeEnv}] → ${config.backendUrl}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  server.close(() => {
    // eslint-disable-next-line no-console
    console.log('BFF server closed');
    process.exit(0);
  });
});

export default server;
