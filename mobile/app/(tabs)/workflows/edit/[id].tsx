// Workflow edit removed from wedge — redirects to inbox
import { Redirect } from 'expo-router';
export default function WorkflowEditRedirect() {
  return <Redirect href="/inbox" />;
}
