// Top-level copilot → /support redirect
// Copilot available as sub-route within support and sales wedge (wave 3/4).
import { Redirect } from 'expo-router';

export default function CopilotRedirect() {
  return <Redirect href="/support" />;
}
