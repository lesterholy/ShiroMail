import { Outlet } from "react-router-dom";
import { ConsoleShell } from "../../../components/layout/console-shell";
import { useSiteName } from "../../../hooks/use-site-name";
import { adminSidebarSections, adminTopNav } from "../../../lib/console-nav";
import { useAuthStore } from "../../../lib/auth-store";

export function AdminConsoleLayout() {
  const user = useAuthStore((state) => state.user);
  const clearSession = useAuthStore((state) => state.clearSession);
  const siteName = useSiteName();

  return (
    <ConsoleShell
      brand={siteName}
      onLogout={clearSession}
      role="admin"
      sections={adminSidebarSections}
      topNav={adminTopNav}
      username={user?.username ?? "Admin"}
    >
      <Outlet />
    </ConsoleShell>
  );
}
