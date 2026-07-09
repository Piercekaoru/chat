"use client";

import * as React from "react";
import dynamic from "next/dynamic";
import { usePathname } from "next/navigation";

import { AuthGuard } from "@/shared/auth/auth-guard";
import { AuthSessionProvider } from "@/shared/auth/auth-session-context";
import { resolveAccessToken } from "@/shared/auth/resolve-access-token";
import { readAccessToken, SESSION_SNAPSHOT_CHANGED_EVENT, type SessionSnapshot } from "@/shared/auth/session";

const AdminAccessGate = dynamic(
  () => import("@/features/admin/components/admin-access-gate").then((mod) => mod.AdminAccessGate),
  { ssr: false },
);

const ProjectLayout = dynamic(
  () => import("@/features/layouts/components/sections/project-layout").then((mod) => mod.ProjectLayout),
  { ssr: false },
);

const GuestChatLanding = dynamic(
  () => import("@/features/chat/components/guest/guest-chat-landing").then((mod) => mod.GuestChatLanding),
  { ssr: false },
);

function isPathWithin(pathname: string, basePath: string): boolean {
  return pathname === basePath || pathname.startsWith(`${basePath}/`);
}

function isPublicPath(pathname: string): boolean {
  return pathname === "/login" || isPathWithin(pathname, "/auth");
}

// Guests may browse only the empty chat landing. The root path redirects to /chat for
// authed users, so it is guest-eligible too. Any conversation/project query param targets
// real data that requires a session, so those fall back to the login redirect.
function isGuestEligibleChatURL(pathname: string): boolean {
  if (pathname !== "/chat" && pathname !== "/") {
    return false;
  }
  if (typeof window === "undefined") {
    return true;
  }
  const params = new URLSearchParams(window.location.search);
  return !params.get("conversation_id")?.trim() && !params.get("project_id")?.trim();
}

function OptionalShareWorkspace({ children }: { children: React.ReactNode }) {
  const [accessToken, setAccessToken] = React.useState<string | null>(() => readAccessToken() || null);

  React.useEffect(() => {
    let cancelled = false;

    async function checkSession() {
      try {
        const token = await resolveAccessToken();
        if (!cancelled) {
          setAccessToken(token || null);
        }
      } catch {
        if (!cancelled) {
          setAccessToken(null);
        }
      }
    }

    void checkSession();
    return () => {
      cancelled = true;
    };
  }, []);

  React.useEffect(() => {
    function handleSessionChanged(event: Event) {
      const snapshot = (event as CustomEvent<SessionSnapshot>).detail;
      const nextToken = snapshot?.accessToken ?? "";
      setAccessToken(nextToken || null);
    }

    window.addEventListener(SESSION_SNAPSHOT_CHANGED_EVENT, handleSessionChanged as EventListener);
    return () => {
      window.removeEventListener(SESSION_SNAPSHOT_CHANGED_EVENT, handleSessionChanged as EventListener);
    };
  }, []);

  if (!accessToken) {
    return <>{children}</>;
  }

  return (
    <AuthSessionProvider accessToken={accessToken}>
      <ProjectLayout defaultSidebarOpen={false}>{children}</ProjectLayout>
    </AuthSessionProvider>
  );
}

type GuestChatWorkspaceStatus = "checking" | "guest" | "authed";

// Renders the empty `/chat` landing for guests and swaps to the authed chat shell in
// place once a session exists. Mirrors OptionalShareWorkspace's token resolution but
// falls back to a guest landing (not the protected children) when there is no session.
function GuestChatWorkspace({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = React.useState<GuestChatWorkspaceStatus>(
    () => (readAccessToken() ? "authed" : "checking"),
  );

  React.useEffect(() => {
    let cancelled = false;

    async function checkSession() {
      let token = "";
      try {
        token = await resolveAccessToken();
      } catch {
        token = "";
      }
      if (!cancelled) {
        setStatus(token ? "authed" : "guest");
      }
    }

    void checkSession();
    return () => {
      cancelled = true;
    };
  }, []);

  React.useEffect(() => {
    function handleSessionChanged(event: Event) {
      const snapshot = (event as CustomEvent<SessionSnapshot>).detail;
      const nextToken = snapshot?.accessToken ?? "";
      setStatus(nextToken ? "authed" : "guest");
    }

    window.addEventListener(SESSION_SNAPSHOT_CHANGED_EVENT, handleSessionChanged as EventListener);
    return () => {
      window.removeEventListener(SESSION_SNAPSHOT_CHANGED_EVENT, handleSessionChanged as EventListener);
    };
  }, []);

  if (status === "authed") {
    return (
      <AuthGuard>
        <ProjectLayout>{children}</ProjectLayout>
      </AuthGuard>
    );
  }

  if (status === "guest") {
    return <GuestChatLanding />;
  }

  // While resolving a possible refresh-token session, render nothing to avoid flashing
  // the guest landing for returning users. AuthGuard shows its own spinner once authed.
  return null;
}

export function WorkspaceShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();

  if (isPublicPath(pathname)) {
    return <>{children}</>;
  }

  if (isPathWithin(pathname, "/share")) {
    return <OptionalShareWorkspace>{children}</OptionalShareWorkspace>;
  }

  if (isGuestEligibleChatURL(pathname)) {
    return <GuestChatWorkspace>{children}</GuestChatWorkspace>;
  }

  const content = isPathWithin(pathname, "/admin")
    ? <AdminAccessGate>{children}</AdminAccessGate>
    : children;

  return (
    <AuthGuard>
      <ProjectLayout>{content}</ProjectLayout>
    </AuthGuard>
  );
}
