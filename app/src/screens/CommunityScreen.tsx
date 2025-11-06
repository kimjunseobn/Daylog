import React, { useEffect, useState } from "react";
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  RefreshControl,
  TouchableOpacity
} from "react-native";
import Card from "../components/Card";
import { useAuth } from "../context/AuthContext";
import { graphqlFetch } from "../api/gateway";
import { AuthHeaders } from "../types/auth";

interface Community {
  id: string;
  title: string;
  description: string;
  is_pro_only: boolean;
  access_level: string;
  created_at: string;
}

const CommunityScreen: React.FC = () => {
  const { user } = useAuth();
  const [communities, setCommunities] = useState<Community[]>([]);
  const [joining, setJoining] = useState<string | null>(null);
  const headers: AuthHeaders = {
    userId: user?.userId ?? "",
    tier: user?.tier ?? "free"
  };

  const fetchCommunities = async () => {
    setJoining(null);
    try {
      const data = await graphqlFetch<{ communities: Community[] }>(
        {
          query: `
            query Communities($includePro: Boolean) {
              communities(includePro: $includePro) {
                id
                title
                description
                is_pro_only
                access_level
                created_at
              }
            }
          `,
          variables: {
            includePro: user?.tier === "pro"
          }
        },
        headers
      );
      setCommunities(data.communities ?? []);
    } catch (error) {
      console.warn("communities fetch failed", error);
    }
  };

  const handleJoin = async (communityId: string) => {
    if (!user) return;
    setJoining(communityId);
    try {
      await graphqlFetch(
        {
          query: `
            mutation JoinCommunity($input: JoinCommunityInput!) {
              joinCommunity(input: $input) {
                community_id
                role
              }
            }
          `,
          variables: {
            input: {
              community_id: communityId,
              user_id: user.userId
            }
          }
        },
        headers
      );
    } catch (error) {
      console.warn("join community failed", error);
    } finally {
      setJoining(null);
    }
  };

  useEffect(() => {
    fetchCommunities();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.tier]);

  return (
    <View style={styles.container}>
      <Text style={styles.title}>커뮤니티</Text>
      <FlatList
        data={communities}
        keyExtractor={(item) => item.id}
        refreshControl={
          <RefreshControl refreshing={joining === null && false} onRefresh={fetchCommunities} />
        }
        renderItem={({ item }) => (
          <Card>
            <Text style={styles.communityTitle}>{item.title}</Text>
            <Text style={styles.description}>{item.description || "소개글이 없습니다."}</Text>
            <View style={styles.metaRow}>
              <Text style={[styles.badge, item.is_pro_only && styles.proBadge]}>
                {item.is_pro_only ? "PRO" : item.access_level.toUpperCase()}
              </Text>
              <TouchableOpacity
                style={styles.joinButton}
                onPress={() => handleJoin(item.id)}
                disabled={joining === item.id}
              >
                <Text style={styles.joinText}>
                  {joining === item.id ? "가입중..." : "참여하기"}
                </Text>
              </TouchableOpacity>
            </View>
          </Card>
        )}
      />
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
  communityTitle: {
    fontSize: 18,
    fontWeight: "700",
    color: "#1B2559",
    marginBottom: 6
  },
  description: {
    fontSize: 15,
    color: "#475467",
    marginBottom: 12
  },
  metaRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center"
  },
  badge: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 999,
    backgroundColor: "#E0E7FF",
    color: "#4C6FFF",
    fontWeight: "600",
    overflow: "hidden"
  },
  proBadge: {
    backgroundColor: "#FFE4D6",
    color: "#FF6B3D"
  },
  joinButton: {
    backgroundColor: "#4C6FFF",
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 12
  },
  joinText: {
    color: "#ffffff",
    fontWeight: "600"
  }
});

export default CommunityScreen;
