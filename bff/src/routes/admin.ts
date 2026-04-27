// BFF-ADMIN-01 / BFF-ADMIN-02: /bff/admin router — shell with shared HTMX chrome and bearer relay
import { Router, Request, Response } from 'express';
import { adminLayout } from './adminLayout';
import { adminBearerRelay } from './adminAuth';
// BFF-ADMIN-10: workflows sub-router
import adminWorkflowsRouter from './adminWorkflows';
// BFF-ADMIN-20: agent runs sub-router
import adminAgentRunsRouter from './adminAgentRuns';
// BFF-ADMIN-30: approvals queue sub-router
import adminApprovalsRouter from './adminApprovals';
// BFF-ADMIN-40: audit trail sub-router
import adminAuditRouter from './adminAudit';
// BFF-ADMIN-50: governance summary + policy sets sub-router
import adminPolicyRouter from './adminPolicy';
// BFF-ADMIN-60: tools list sub-router
import adminToolsRouter from './adminTools';
// BFF-ADMIN-70: metrics dashboard sub-router
import adminMetricsRouter from './adminMetrics';

const router = Router();

// BFF-ADMIN-02: attach bearer token to res.locals for all admin proxy handlers (Phase B+)
router.use(adminBearerRelay);

const dashboardBody = `
  <h2 class="page-title">Dashboard</h2>
  <p class="placeholder">dashboard — phase B–H pages will populate counts and recent runs here.</p>
`;

router.get(['/', '/dashboard'], (_req: Request, res: Response): void => {
  res.type('html').status(200).send(adminLayout('Dashboard', dashboardBody));
});

// BFF-ADMIN-10
router.use('/workflows', adminWorkflowsRouter);
// BFF-ADMIN-20
router.use('/agent-runs', adminAgentRunsRouter);
// BFF-ADMIN-30
router.use('/approvals', adminApprovalsRouter);
// BFF-ADMIN-40
router.use('/audit', adminAuditRouter);
// BFF-ADMIN-50
router.use('/policy', adminPolicyRouter);
// BFF-ADMIN-60
router.use('/tools', adminToolsRouter);
// BFF-ADMIN-70
router.use('/metrics', adminMetricsRouter);

export default router;
