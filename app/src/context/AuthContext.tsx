import React, { createContext, useContext, useMemo, useState } from "react";

export type UserTier = "free" | "pro";

export interface UserSession {
  userId: string;
  tier: UserTier;
  labels: Record<string, string>;
}

interface AuthContextValue {
  user: UserSession | null;
  login: (session: UserSession) => void;
  logout: () => void;
  updateTier: (tier: UserTier) => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<UserSession | null>(null);

  const value = useMemo(
    () => ({
      user,
      login: (session: UserSession) => setUser(session),
      logout: () => setUser(null),
      updateTier: (tier: UserTier) =>
        setUser((prev) => (prev ? { ...prev, tier } : prev))
    }),
    [user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
};
