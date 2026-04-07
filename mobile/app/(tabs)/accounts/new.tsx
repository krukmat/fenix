// Account creation removed from wedge — redirects to sales hub
import { Redirect } from 'expo-router';
export default function AccountNewRedirect() {
  return <Redirect href="/sales" />;
}
