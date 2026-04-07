// W2-T2 (mobile_wedge_harmonization_plan): /home → /inbox redirect
// Home feed migrated to the Inbox wedge tab.
import { Redirect } from 'expo-router';

export default function HomeRedirect() {
  return <Redirect href="/inbox" />;
}
