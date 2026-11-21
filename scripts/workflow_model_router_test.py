"""Pure offline tests for opt-in model routing: no Ollama invocation."""
import unittest
import workflow_model_router as router


class ModelRoutingTest(unittest.TestCase):
    def setUp(self):
        self.params = {
            "role": "fix", "rri": 15, "paths": ("scripts/sample.py",),
            "task_id": "T-1", "environment": {"FENIX_ROLE_FIX_MODEL": "model:tag"},
        }

    def test_eligible_role(self):
        self.assertEqual(router.configured_low_rri_model(**self.params), "model:tag")

    def test_no_implicit_model_fallback(self):
        self.params["environment"] = {}
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_moderate_rejected_even_with_model(self):
        self.params["rri"] = 26
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_negative_rri_rejected(self):
        self.params["rri"] = -1
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_bool_rri_rejected(self):
        self.params["rri"] = True
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_protected_path_rejected(self):
        self.params["paths"] = ("docs/policies/RRI_POLICY.md",)
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_peer_review_script_protected(self):
        self.params["paths"] = ("scripts/peer-workflow-review.py",)
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_traversal_rejected(self):
        self.params["paths"] = ("scripts/../internal/x.py",)
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_glob_rejected(self):
        self.params["paths"] = ("scripts/*.py",)
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_invalid_role_rejected(self):
        self.params["role"] = "precheck"
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_task_id_required(self):
        self.params["task_id"] = ""
        with self.assertRaises(router.RoutingBlocked):
            router.configured_low_rri_model(**self.params)

    def test_precheck_is_never_approval(self):
        out = router.advisory_precheck({"status": "PASS", "reason": "structured", "evidence_refs": []})
        self.assertTrue(out["valid"])
        self.assertTrue(out["peer_review_required"])

    def test_precheck_invalid_escalates(self):
        self.assertEqual(router.advisory_precheck({"status": "PASS"})["status"], "ESCALATE")




class ModerateAdvisoryTest(unittest.TestCase):
    def setUp(self):
        self.params = dict(task_id="T-26", rri=30, paths=["scripts/sample.py"],
                           human_approved=True, critical=False,
                           has_acceptance_criteria=True, has_relevant_tests=True)

    def test_policy_not_adopted_always_denies(self):
        result = router.moderate_delegation_advisory(**self.params)
        self.assertFalse(result["eligible"])
        self.assertEqual(result["reason"], "active_policy_forbids_rri_26_plus")

    def test_approval_cannot_replace_criteria(self):
        self.params["has_relevant_tests"] = False
        result = router.moderate_delegation_advisory(governance_policy_adopted=True, **self.params)
        self.assertFalse(result["eligible"])

    def test_critical_denied(self):
        self.params["critical"] = True
        result = router.moderate_delegation_advisory(governance_policy_adopted=True, **self.params)
        self.assertFalse(result["eligible"])

    def test_protected_path_denied(self):
        self.params["paths"] = ["scripts/peer-workflow-review.py"]
        result = router.moderate_delegation_advisory(governance_policy_adopted=True, **self.params)
        self.assertFalse(result["eligible"])

    def test_conforming_future_case_remains_advisory_only(self):
        result = router.moderate_delegation_advisory(governance_policy_adopted=True, **self.params)
        self.assertTrue(result["eligible"])
        self.assertIn("advisory_only", result["reason"])


if __name__ == "__main__":
    unittest.main()
