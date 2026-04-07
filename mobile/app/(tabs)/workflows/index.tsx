// Workflows index → /inbox redirect
// Workflow nav removed from visible shell. Full removal pending wave 6.
import { Redirect } from 'expo-router';

export default function WorkflowsRedirect() {
  return <Redirect href="/inbox" />;
}
