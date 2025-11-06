import React, { useEffect, useState } from "react";
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  RefreshControl,
  TextInput,
  TouchableOpacity
} from "react-native";
import Card from "../components/Card";
import { useAuth } from "../context/AuthContext";
import { graphqlFetch } from "../api/gateway";
import { AuthHeaders } from "../types/auth";

interface FeedItem {
  post_id: string;
  user_id: string;
  timeline_id: string;
  category: string;
  message: string;
  created_at: string;
}

const FeedScreen: React.FC = () => {
  const { user } = useAuth();
  const [feed, setFeed] = useState<FeedItem[]>([]);
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const headers: AuthHeaders = {
    userId: user?.userId ?? "",
    tier: user?.tier ?? "free"
  };

  const fetchFeed = async () => {
    if (!user) return;
    setLoading(true);
    try {
      const data = await graphqlFetch<{ feed: FeedItem[] }>(
        {
          query: `
            query Feed($userId: ID!) {
              feed(userId: $userId, limit: 50) {
                post_id
                category
                message
                created_at
                user_id
              }
            }
          `,
          variables: { userId: user.userId }
        },
        headers
      );
      setFeed(data.feed ?? []);
    } catch (error) {
      console.warn("feed fetch failed", error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreatePost = async () => {
    if (!user || !message.trim()) return;
    try {
      await graphqlFetch<{ createFeedPost: FeedItem }>(
        {
          query: `
            mutation CreatePost($input: CreateFeedPostInput!) {
              createFeedPost(input: $input) {
                post_id
                category
                message
                created_at
                user_id
              }
            }
          `,
          variables: {
            input: {
              user_id: user.userId,
              timeline_id: "manual-entry",
              category: "reflection",
              message
            }
          }
        },
        headers
      );
      setMessage("");
      fetchFeed();
    } catch (error) {
      console.warn("create post failed", error);
    }
  };

  useEffect(() => {
    fetchFeed();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.userId]);

  return (
    <View style={styles.container}>
      <View style={styles.postBox}>
        <TextInput
          style={styles.input}
          placeholder="오늘 하루를 한 줄로 기록해 보세요."
          value={message}
          onChangeText={setMessage}
          multiline
        />
        <TouchableOpacity style={styles.submit} onPress={handleCreatePost}>
          <Text style={styles.submitText}>공유하기</Text>
        </TouchableOpacity>
      </View>

      <FlatList
        data={feed}
        keyExtractor={(item) => item.post_id}
        refreshControl={
          <RefreshControl refreshing={loading} onRefresh={fetchFeed} />
        }
        renderItem={({ item }) => (
          <Card>
            <Text style={styles.category}>#{item.category}</Text>
            <Text style={styles.message}>{item.message}</Text>
            <Text style={styles.meta}>
              {item.user_id === user?.userId ? "나" : item.user_id.slice(0, 8)} ·{" "}
              {new Date(item.created_at).toLocaleString("ko-KR", {
                month: "short",
                day: "numeric",
                hour: "2-digit",
                minute: "2-digit"
              })}
            </Text>
          </Card>
        )}
        ListEmptyComponent={
          !loading ? (
            <View style={styles.empty}>
              <Text style={styles.emptyText}>피드가 아직 비어 있습니다.</Text>
            </View>
          ) : null
        }
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
  postBox: {
    backgroundColor: "#ffffff",
    borderRadius: 16,
    padding: 16,
    marginBottom: 16,
    shadowColor: "#101828",
    shadowOpacity: 0.08,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 6 },
    elevation: 2
  },
  input: {
    minHeight: 80,
    borderWidth: 1,
    borderColor: "#E0E7FF",
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 10,
    marginBottom: 12,
    fontSize: 16
  },
  submit: {
    backgroundColor: "#4C6FFF",
    paddingVertical: 12,
    borderRadius: 12,
    alignItems: "center"
  },
  submitText: {
    color: "#ffffff",
    fontSize: 15,
    fontWeight: "700"
  },
  category: {
    fontSize: 14,
    color: "#4C6FFF",
    fontWeight: "600",
    marginBottom: 6
  },
  message: {
    fontSize: 16,
    color: "#1B2559",
    marginBottom: 6
  },
  meta: {
    fontSize: 12,
    color: "#98A2B3"
  },
  empty: {
    alignItems: "center",
    marginTop: 64
  },
  emptyText: {
    fontSize: 16,
    color: "#98A2B3"
  }
});

export default FeedScreen;
