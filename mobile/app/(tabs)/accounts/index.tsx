// W2-T2 (mobile_wedge_harmonization_plan): /accounts → /sales redirect
// Account browsing migrated to the Sales wedge tab (W4-T1).
import { Redirect } from 'expo-router';

export default function AccountsRedirect() {
  return <Redirect href="/sales" />;
}
