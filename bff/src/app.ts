// Task 4.1 — FR-301: Express app factory — exported without listen() for Supertest compatibility
import express from 'express';
import helmet from 'helmet';
import cors from 'cors';
import { config } from './config';

import { authRelay } from './middleware/authRelay';
import { mobileHeaders } from './middleware/mobileHeaders';
import { errorHandler } from './middleware/errorHandler';

import healthRouter from './routes/health';
import authRouter from './routes/auth';
import proxyRouter from './routes/proxy';
import copilotRouter from './routes/copilot';
import metricsRouter, { incRequests } from './routes/metrics';
// W1-T2: approval alias routes (approve/reject)
import approvalsRouter from './routes/approvals';
// W1-T3: inbox aggregation route
import inboxRouter from './routes/inbox';
import builderRouter from './routes/builder';

const app = express();

// Security middleware
app.use(helmet());
app.use(cors({
  origin(origin, callback) {
    if (!origin || config.corsAllowedOrigins.includes(origin)) {
      callback(null, true);
      return;
    }
    callback(null, false);
  },
}));

// Body parsing
app.use(express.json());
app.use(express.urlencoded({ extended: false }));

// Mobile header extraction (before any route)
app.use(mobileHeaders);

// Auth relay (before any route that needs tokens)
app.use(authRelay);

// Task 4.9 — NFR-030: count all requests for metrics
app.use((_req, _res, next) => {
  incRequests();
  next();
});

// Routes
app.use('/bff/health', healthRouter);
app.use('/bff/metrics', metricsRouter);
app.use('/bff/auth', authRouter);
app.use('/bff/copilot', copilotRouter);
app.use('/bff/builder', builderRouter);

// F4-T2: JSON copilot routes that must bypass the transparent proxy
app.use('/bff/api/v1/copilot', copilotRouter);

// W1-T2: approval alias routes — must be registered BEFORE the transparent proxy
// so /bff/api/v1/approvals/:id/approve and /reject are handled here, not proxied
app.use('/bff/api/v1/approvals', approvalsRouter);

// W1-T3: inbox aggregation — before transparent proxy
app.use('/bff/api/v1/mobile/inbox', inboxRouter);

// Transparent proxy for all other /bff/api/v1/* calls
app.use('/bff/api/v1', proxyRouter);

// Error handler (must be last)
app.use(errorHandler);

export default app;
