// CRM hub → /sales redirect
// CRM hub replaced by Sales wedge tab. Full removal pending wave 6.
import { Redirect } from 'expo-router';

export default function CrmRedirect() {
  return <Redirect href="/sales" />;
}
