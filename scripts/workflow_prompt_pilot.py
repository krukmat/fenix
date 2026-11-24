"""Offline W6 readiness prompt pilot; no provider calls or raw prompt export.

The pinned historical fixture compares plan consolidation, then stable layout.
These are descriptive bytes, not acceptance or provider-token evidence.
"""
import ast
import json
from pathlib import Path

import workflow_prompt_layout as layout

BASELINE = 'c5c366364ac1f44ce2fff386731d166614a66701'
CANDIDATE = 'd9cd010b10d5203f59358f9a4c20cc826147c88e'
PLAN = 'docs/plans/ai_workflow_w6_cloud_local_execution_plan.md'
TASK = 'docs/tasks/task_fenix_ai_workflow_w6_%d.md'
EVIDENCE = (
    ('wire contract', 'docs/ai-workflow/cloud_local_wire_contract_v1_draft.md'),
    ('benchmark contract', 'docs/ai-workflow/cloud_local_integration_benchmark_draft.md'),
    ('HITL policy', 'docs/policies/HITL_AUTONOMY_POLICY.md'),
    ('RRI policy', 'docs/policies/RRI_POLICY.md'),
)
NATIVE = 'scripts/peer-workflow-review.py'


def load_renderer(source):
    """Extract only trusted, pinned native renderer definitions, not CLI execution."""
    tree = ast.parse(source)
    constants = {'REVIEW_INSTRUCTIONS', 'TASK_READINESS_CRITICALITY_INSTRUCTIONS'}
    nodes = [n for n in tree.body if
             (isinstance(n, ast.Assign) and len(n.targets) == 1 and
              isinstance(n.targets[0], ast.Name) and n.targets[0].id in constants) or
             (isinstance(n, ast.FunctionDef) and n.name == 'build_prompt')]
    if len(nodes) != 3 or {n.targets[0].id for n in nodes if isinstance(n, ast.Assign)} != constants:
        raise ValueError('native renderer definitions unavailable')
    ns = {}
    exec(compile(ast.Module(body=nodes, type_ignores=[]), 'pinned_native_renderer', 'exec'), ns)
    return ns['build_prompt']


def prepare_cases(read_blob):
    """Return raw prompts in memory only; callers must authorize any disclosure.

    read_blob(commit, path) must read the exact immutable, trusted Git blob.
    The historical evidence bundle is a size fixture, not complete live authority.
    """
    def read(commit, path):
        data = read_blob(commit, path)
        if type(data) is not bytes:
            raise ValueError('immutable bytes required')
        return data.decode('utf-8')
    renderer = load_renderer(read(CANDIDATE, NATIVE))
    historical = read(BASELINE, PLAN)
    compact = read(CANDIDATE, PLAN)
    evidence = '\n'.join('=== %s ===\n%s' % (name, read(CANDIDATE, path)) for name, path in EVIDENCE)
    cases = []
    for n in range(1, 10):
        task = read(CANDIDATE, TASK % n)
        base = dict(mode='task-readiness', task=task, plan=historical, task_card=evidence)
        candidate = dict(base, plan=compact)
        variants = {}
        for name, packet, order in (('historical', base, 'native'),
                                    ('compact', candidate, 'native'),
                                    ('prefix', candidate, 'stable_prefix')):
            variants[name] = layout.prepare(packet, renderer, layout=order)
        if variants['compact'][1]['section_sha256'] != variants['prefix'][1]['section_sha256']:
            raise ValueError('layout content mismatch')
        for key in ('task', 'task_card'):
            if variants['historical'][1]['section_sha256'][key] != variants['compact'][1]['section_sha256'][key]:
                raise ValueError('non-plan evidence changed')
        cases.append((f'W6.{n}', variants))
    return cases


def report(read_blob):
    cases = prepare_cases(read_blob)
    arms = ('historical', 'compact', 'prefix')
    return dict(schema='fenix.prompt-pilot-preflight.v1', baseline_commit=BASELINE,
                candidate_commit=CANDIDATE, case_count=len(cases), inference_invoked=False,
                cases=[dict(case_id=case_id, arms={name: variants[name][1] for name in arms})
                       for case_id, variants in cases],
                common_prefix_bytes={name: layout.common_prefix_bytes([v[name][0] for _,v in cases]) for name in arms},
                limitations=['Historical size fixture only; confirm all current required evidence before any live pilot.',
                             'Bytes and shared prefixes are not model token counts or cache-hit evidence.',
                             'No peer verdict, formal gate, authorization, or provider call.'])


def main():
    from workflow_context import read_git_blob
    root = Path(__file__).resolve().parent.parent
    result = report(lambda commit, path: read_git_blob(root, commit, path, max_bytes=1048576))
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()
