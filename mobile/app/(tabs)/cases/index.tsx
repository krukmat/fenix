// W2-T2 (mobile_wedge_harmonization_plan): /cases → /support redirect
// Case browsing migrated to the Support wedge tab (W3-T1).
import { Redirect } from 'expo-router';

export default function CasesRedirect() {
  return <Redirect href="/support" />;
}
