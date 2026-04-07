// Workflow creation removed from wedge — redirects to inbox
import { Redirect } from 'expo-router';
export default function WorkflowNewRedirect() {
  return <Redirect href="/inbox" />;
}
