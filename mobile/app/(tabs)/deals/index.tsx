// W2-T2 (mobile_wedge_harmonization_plan): /deals → /sales redirect
// Deal browsing migrated to the Sales wedge tab (W4-T1).
import { Redirect } from 'expo-router';

export default function DealsRedirect() {
  return <Redirect href="/sales" />;
}
