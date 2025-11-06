import { UserTier } from "../context/AuthContext";

export interface AuthHeaders {
  userId: string;
  tier: UserTier;
}
