import { reactive } from 'vue';
import { api, type Me, type SignInOptions } from './api';

// Who is signed in, shared by every view.
export const session = reactive<{ me: Me | null; options: SignInOptions | null; loaded: boolean }>({
  me: null,
  options: null,
  loaded: false,
});

export async function loadSession(): Promise<Me | null> {
  const s = await api.session();
  session.me = s.signedIn ? { role: s.role, login: s.login, expiresAt: s.expiresAt } : null;
  session.loaded = true;
  return session.me;
}

export async function signOut() {
  await api.logout();
  session.me = null;
}
