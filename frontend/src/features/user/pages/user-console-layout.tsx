import { Outlet } from "react-router-dom";
import { ConsoleShell } from "../../../components/layout/console-shell";
import { useSiteName } from "../../../hooks/use-site-name";
import { userSidebarSections, userTopNav } from "../../../lib/console-nav";
import { useAuthStore } from "../../../lib/auth-store";

export function UserConsoleLayout() {
  const user = useAuthStore((state) => state.user);
  const clearSession = useAuthStore((state) => state.clearSession);
  const siteName = useSiteName();

  return (
    <ConsoleShell
      brand={siteName}
      onLogout={clearSession}
      role="user"
      sections={userSidebarSections}
      topNav={userTopNav}
      username={user?.username ?? "User"}
    >
      <Outlet />
    </ConsoleShell>
  );
}
