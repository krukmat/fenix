// Contacts → /sales redirect
// Top-level contacts removed from visible nav. Full removal pending wave 6.
import { Redirect } from 'expo-router';

export default function ContactsRedirect() {
  return <Redirect href="/sales" />;
}
