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


if __name__ == "__main__":
    unittest.main()
