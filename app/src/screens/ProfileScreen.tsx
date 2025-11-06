import React, { useEffect, useState } from "react";
import { View, Text, StyleSheet, TouchableOpacity, Alert } from "react-native";
import Card from "../components/Card";
import { useAuth } from "../context/AuthContext";
import { graphqlFetch } from "../api/gateway";
import { AuthHeaders } from "../types/auth";

interface Entitlement {
  user_id: string;
  tier: string;
  status: string;
  renewal_date?: string;
  stripe_subscription_id?: string;
}

const ProfileScreen: React.FC = () => {
  const { user, logout, updateTier } = useAuth();
  const [entitlement, setEntitlement] = useState<Entitlement | null>(null);
  const [loading, setLoading] = useState(false);

  const headers: AuthHeaders = {
    userId: user?.userId ?? "",
    tier: user?.tier ?? "free"
  };

  const fetchEntitlement = async () => {
    if (!user) return;
    setLoading(true);
    try {
      const data = await graphqlFetch<{ viewerEntitlement: Entitlement | null }>(
        {
          query: `
            query ViewerEntitlement {
              viewerEntitlement {
                user_id
                tier
                status
                renewal_date
                stripe_subscription_id
              }
            }
          `
        },
        headers
      );
      if (data.viewerEntitlement) {
        setEntitlement(data.viewerEntitlement);
        updateTier((data.viewerEntitlement.tier as "pro" | "free") ?? user.tier);
      }
    } catch (error) {
      console.warn("entitlement fetch failed", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEntitlement();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleUpgradeInfo = () => {
    Alert.alert(
      "결제 안내",
      "Stripe 테스트 대시보드에서 구독 상태를 변경하면 여기에 반영됩니다. Pro 전용 기능을 체험하려면 tier를 pro로 유지하세요."
    );
  };

  if (!user) {
    return null;
  }

  return (
    <View style={styles.container}>
      <Text style={styles.title}>내 정보</Text>
      <Card>
        <Text style={styles.label}>User ID</Text>
        <Text style={styles.value}>{user.userId}</Text>

        <Text style={styles.label}>Current Tier</Text>
        <Text style={[styles.value, user.tier === "pro" && styles.proValue]}>
          {user.tier.toUpperCase()}
        </Text>

        {entitlement && (
          <>
            <Text style={styles.label}>Entitlement Status</Text>
            <Text style={styles.value}>{entitlement.status}</Text>
            {entitlement.renewal_date && (
              <>
                <Text style={styles.label}>Renewal Date</Text>
                <Text style={styles.value}>
                  {new Date(entitlement.renewal_date).toLocaleString("ko-KR")}
                </Text>
              </>
            )}
          </>
        )}

        <TouchableOpacity style={styles.button} onPress={fetchEntitlement} disabled={loading}>
          <Text style={styles.buttonText}>
            {loading ? "새로고침 중..." : "구독 정보 새로고침"}
          </Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.secondary} onPress={handleUpgradeInfo}>
          <Text style={styles.secondaryText}>Pro 업그레이드 안내</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.logout} onPress={logout}>
          <Text style={styles.logoutText}>로그아웃</Text>
        </TouchableOpacity>
      </Card>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#F4F7FB",
    paddingHorizontal: 16,
    paddingTop: 20
  },
  title: {
    fontSize: 24,
    fontWeight: "700",
    color: "#1B2559",
    marginBottom: 16
  },
  label: {
    fontSize: 13,
    color: "#98A2B3",
    textTransform: "uppercase",
    marginTop: 12
  },
  value: {
    fontSize: 16,
    color: "#1B2559",
    fontWeight: "600",
    marginTop: 4
  },
  proValue: {
    color: "#FF6B3D"
  },
  button: {
    backgroundColor: "#4C6FFF",
    paddingVertical: 12,
    borderRadius: 12,
    alignItems: "center",
    marginTop: 20
  },
  buttonText: {
    color: "#ffffff",
    fontWeight: "700"
  },
  secondary: {
    paddingVertical: 12,
    borderRadius: 12,
    alignItems: "center",
    marginTop: 12,
    borderWidth: 1,
    borderColor: "#D0D5DD"
  },
  secondaryText: {
    color: "#4C6FFF",
    fontWeight: "600"
  },
  logout: {
    paddingVertical: 12,
    borderRadius: 12,
    alignItems: "center",
    marginTop: 12
  },
  logoutText: {
    color: "#98A2B3",
    fontWeight: "600"
  }
});

export default ProfileScreen;
