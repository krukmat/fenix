#!/usr/bin/env node

/**
 * Seed reusable UAT data for mobile through BFF endpoints.
 *
 * Usage:
 *   node scripts/seed_uat_mobile.mjs
 *   node scripts/seed_uat_mobile.mjs --email test@fenix.local --password 'Password123!'
 *
 * Env:
 *   API_URL=http://localhost:8080
 */

const args = process.argv.slice(2);

function argValue(name, fallback = '') {
  const idx = args.indexOf(name);
  if (idx === -1 || idx + 1 >= args.length) return fallback;
  return args[idx + 1];
}

const now = Date.now();
const apiUrl = process.env.API_URL || 'http://localhost:8080';
const email = argValue('--email', `uat.mobile.seed.${now}@fenix.local`);
const password = argValue('--password', 'Password123!');
const displayName = argValue('--display-name', 'UAT Mobile Seed');
const workspaceName = argValue('--workspace', 'UAT Workspace');

async function requestJson(method, path, body, token) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${apiUrl}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  let data = null;
  try {
    data = await res.json();
  } catch (_err) {
    data = null;
  }

  return { ok: res.ok, status: res.status, data };
}

function must(ok, message, detail) {
  if (!ok) {
    const err = new Error(message);
    err.detail = detail;
    throw err;
  }
}

async function loginOrRegister() {
  const loginRes = await requestJson('POST', '/auth/login', { email, password });
  if (loginRes.ok) return loginRes.data;

  if (loginRes.status !== 401) {
    throw new Error(`Login failed with status ${loginRes.status}: ${JSON.stringify(loginRes.data)}`);
  }

  const registerRes = await requestJson('POST', '/auth/register', {
    displayName,
    email,
    password,
    workspaceName,
  });
  must(registerRes.ok, `Register failed with status ${registerRes.status}`, registerRes.data);
  return registerRes.data;
}

async function seedPipeline(token) {
  // Create a default Sales Pipeline with 3 stages (required for deal creation)
  const pipelineRes = await requestJson(
    'POST',
    '/api/v1/pipelines',
    { name: 'Sales Pipeline', entityType: 'deal' },
    token,
  );
  must(pipelineRes.ok, `Create pipeline failed (${pipelineRes.status})`, pipelineRes.data);
  const pipeline = pipelineRes.data;

  const stageNames = ['Prospecting', 'Qualified', 'Closed Won'];
  const createdStages = [];
  for (let i = 0; i < stageNames.length; i += 1) {
    const stageRes = await requestJson(
      'POST',
      `/api/v1/pipelines/${pipeline.id}/stages`,
      { name: stageNames[i], position: i + 1 },
      token,
    );
    must(stageRes.ok, `Create stage failed (${stageRes.status})`, stageRes.data);
    createdStages.push(stageRes.data);
  }

  return { pipeline, stages: createdStages };
}

async function seed() {
  const auth = await loginOrRegister();
  const token = auth.token;
  const ownerId = auth.userId;
  must(!!token && !!ownerId, 'Auth response missing token/userId', auth);

  // Seed pipeline + stages first — deals depend on them
  const { pipeline, stages } = await seedPipeline(token);

  const accountNames = ['Acme Corp', 'Globex LLC', 'Soylent Inc', 'Initech', 'Umbrella Co'];
  const dealTitles = ['Enterprise License', 'Pro Plan Renewal', 'Consulting Project', 'Support Contract', 'Platform Upgrade'];
  const dealAmounts = [50000, 12000, 35000, 8000, 95000];
  const createdAccounts = [];
  const createdContacts = [];
  const createdCases = [];
  const createdDeals = [];

  for (let i = 0; i < accountNames.length; i += 1) {
    const accountName = `${accountNames[i]} ${now.toString().slice(-5)}`;
    const accountRes = await requestJson(
      'POST',
      '/api/v1/accounts',
      {
        name: accountName,
        industry: i % 2 === 0 ? 'Technology' : 'Services',
        ownerId,
      },
      token,
    );
    must(accountRes.ok, `Create account failed (${accountRes.status})`, accountRes.data);
    const account = accountRes.data;
    createdAccounts.push(account);

    const contactRes = await requestJson(
      'POST',
      '/api/v1/contacts',
      {
        accountId: account.id,
        firstName: `Contact${i + 1}`,
        lastName: 'UAT',
        email: `contact${i + 1}.${now}@fenix.local`,
        phone: `+1-555-100${i}`,
        title: i % 2 === 0 ? 'Manager' : 'Director',
        ownerId,
      },
      token,
    );
    must(contactRes.ok, `Create contact failed (${contactRes.status})`, contactRes.data);
    const contact = contactRes.data;
    createdContacts.push(contact);

    const caseRes = await requestJson(
      'POST',
      '/api/v1/cases',
      {
        accountId: account.id,
        contactId: contact.id,
        ownerId,
        subject: `Support case ${i + 1} - ${accountName}`,
        priority: i % 2 === 0 ? 'high' : 'medium',
        status: 'open',
        channel: 'email',
        description: 'UAT seeded case for mobile validation',
      },
      token,
    );
    must(caseRes.ok, `Create case failed (${caseRes.status})`, caseRes.data);
    createdCases.push(caseRes.data);

    // Each account gets one deal — distributed across the pipeline stages
    const stageIdx = i % stages.length;
    const dealRes = await requestJson(
      'POST',
      '/api/v1/deals',
      {
        title: dealTitles[i],
        amount: dealAmounts[i],
        accountId: account.id,
        pipelineId: pipeline.id,
        stageId: stages[stageIdx].id,
        ownerId,
      },
      token,
    );
    must(dealRes.ok, `Create deal failed (${dealRes.status})`, dealRes.data);
    createdDeals.push(dealRes.data);
  }

  // eslint-disable-next-line no-console
  console.log(
    JSON.stringify(
      {
        apiUrl,
        credentials: { email, password },
        auth: { userId: ownerId, workspaceId: auth.workspaceId },
        created: {
          accounts: createdAccounts.length,
          contacts: createdContacts.length,
          cases: createdCases.length,
          deals: createdDeals.length,
          pipeline: pipeline.name,
          stages: stages.length,
        },
        sampleIds: {
          accountId: createdAccounts[0]?.id,
          contactId: createdContacts[0]?.id,
          caseId: createdCases[0]?.id,
          dealId: createdDeals[0]?.id,
          pipelineId: pipeline.id,
          stageId: stages[0]?.id,
        },
      },
      null,
      2,
    ),
  );
}

seed().catch((err) => {
  // eslint-disable-next-line no-console
  console.error(
    JSON.stringify(
      {
        error: err.message,
        detail: err.detail ?? null,
      },
      null,
      2,
    ),
  );
  process.exit(1);
});
