// Task 4.1 — FR-301: BFF entry point — starts HTTP server
import { validateConfig, config } from './config';

validateConfig();

import app from './app';

const server = app.listen(config.port, () => {
  console.log(`FenixCRM BFF running on port ${config.port}`);
  console.log(`Backend URL: ${config.backendUrl}`);
  console.log(`Environment: ${config.nodeEnv}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  server.close(() => {
    console.log('BFF server closed');
    process.exit(0);
  });
});

export default server;
