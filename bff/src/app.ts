// Task 4.1 — FR-301: Express app factory — exported without listen() for Supertest compatibility
import express from 'express';
import helmet from 'helmet';
import cors from 'cors';

import { authRelay } from './middleware/authRelay';
import { mobileHeaders } from './middleware/mobileHeaders';
import { errorHandler } from './middleware/errorHandler';

import healthRouter from './routes/health';
import authRouter from './routes/auth';
import proxyRouter from './routes/proxy';
import aggregatedRouter from './routes/aggregated';
import copilotRouter from './routes/copilot';

const app = express();

// Security middleware
app.use(helmet());
app.use(cors());

// Body parsing
app.use(express.json());

// Mobile header extraction (before any route)
app.use(mobileHeaders);

// Auth relay (before any route that needs tokens)
app.use(authRelay);

// Routes
app.use('/bff/health', healthRouter);
app.use('/bff/auth', authRouter);
app.use('/bff/copilot', copilotRouter);

// Aggregated routes (before proxy to avoid being caught by proxy wildcard)
app.use('/bff', aggregatedRouter);

// Transparent proxy for all other /bff/api/v1/* calls
app.use('/bff/api/v1', proxyRouter);

// Error handler (must be last)
app.use(errorHandler);

export default app;
