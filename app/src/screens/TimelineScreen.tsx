import React, { useEffect, useState } from "react";
import { View, Text, StyleSheet, FlatList, RefreshControl } from "react-native";
import Card from "../components/Card";
import { useAuth } from "../context/AuthContext";
import { graphqlFetch } from "../api/gateway";
import { AuthHeaders } from "../types/auth";

interface TimelineEntry {
  event_id: string;
  category: string;
  started_at: string;
  ended_at: string;
  confidence: number;
  source: string;
  metadata?: Record<string, unknown>;
}

const TimelineScreen: React.FC = () => {
  const { user } = useAuth();
  const [entries, setEntries] = useState<TimelineEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const headers: AuthHeaders = {
    userId: user?.userId ?? "",
    tier: user?.tier ?? "free"
  };

  const fetchTimeline = async () => {
    if (!user) return;
    setLoading(true);
    try {
      const data = await graphqlFetch<{ timeline: TimelineEntry[] }>(
        {
          query: `
            query Timeline($userId: ID!) {
              timeline(userId: $userId, limit: 25) {
                event_id
                category
                started_at
                ended_at
                confidence
                source
              }
            }
          `,
          variables: { userId: user.userId }
        },
        headers
      );
      setEntries(data.timeline ?? []);
    } catch (error) {
      console.warn("timeline fetch failed", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTimeline();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.userId]);

  return (
    <View style={styles.container}>
      <Text style={styles.title}>오늘의 타임라인</Text>
      <FlatList
        data={entries}
        keyExtractor={(item) => item.event_id}
        refreshControl={
          <RefreshControl refreshing={loading} onRefresh={fetchTimeline} />
        }
        renderItem={({ item }) => (
          <Card>
            <Text style={styles.category}>{item.category}</Text>
            <Text style={styles.time}>
              {formatTime(item.started_at)} - {formatTime(item.ended_at)}
            </Text>
            <Text style={styles.meta}>Source: {item.source}</Text>
            <Text style={styles.meta}>Confidence: {(item.confidence * 100).toFixed(0)}%</Text>
          </Card>
        )}
        ListEmptyComponent={
          !loading ? (
            <View style={styles.empty}>
              <Text style={styles.emptyText}>타임라인 데이터가 없습니다.</Text>
            </View>
          ) : null
        }
      />
    </View>
  );
};

function formatTime(iso: string) {
  try {
    const date = new Date(iso);
    return date.toLocaleTimeString("ko-KR", { hour: "2-digit", minute: "2-digit" });
  } catch {
    return iso;
  }
}

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
  category: {
    fontSize: 18,
    fontWeight: "700",
    color: "#344054",
    textTransform: "capitalize",
    marginBottom: 6
  },
  time: {
    fontSize: 16,
    color: "#475467",
    marginBottom: 4
  },
  meta: {
    fontSize: 14,
    color: "#98A2B3"
  },
  empty: {
    alignItems: "center",
    marginTop: 48
  },
  emptyText: {
    fontSize: 16,
    color: "#98A2B3"
  }
});

export default TimelineScreen;
