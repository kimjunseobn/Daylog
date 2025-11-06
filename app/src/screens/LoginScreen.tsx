import React, { useState } from "react";
import { View, Text, TextInput, StyleSheet, TouchableOpacity } from "react-native";
import { useAuth, UserTier } from "../context/AuthContext";

const LoginScreen: React.FC = () => {
  const { login } = useAuth();
  const [userId, setUserId] = useState("00000000-0000-0000-0000-000000000000");
  const [tier, setTier] = useState<UserTier>("pro");

  const handleLogin = () => {
    if (!userId.trim()) return;
    login({
      userId: userId.trim(),
      tier,
      labels: {}
    });
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Daylog 로그인</Text>
      <Text style={styles.subtitle}>테스트용 사용자 정보를 입력하세요.</Text>
      <View style={styles.form}>
        <Text style={styles.label}>User ID</Text>
        <TextInput
          style={styles.input}
          value={userId}
          onChangeText={setUserId}
          placeholder="UUID"
          autoCapitalize="none"
        />

        <Text style={styles.label}>Tier</Text>
        <View style={styles.tierRow}>
          <TouchableOpacity
            style={[styles.tierButton, tier === "free" && styles.tierSelected]}
            onPress={() => setTier("free")}
          >
            <Text style={tier === "free" ? styles.tierSelectedText : styles.tierText}>
              Free
            </Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.tierButton, tier === "pro" && styles.tierSelected]}
            onPress={() => setTier("pro")}
          >
            <Text style={tier === "pro" ? styles.tierSelectedText : styles.tierText}>
              Pro
            </Text>
          </TouchableOpacity>
        </View>

        <TouchableOpacity style={styles.submit} onPress={handleLogin}>
          <Text style={styles.submitText}>시작하기</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#F4F7FB",
    justifyContent: "center",
    paddingHorizontal: 24
  },
  title: {
    fontSize: 28,
    fontWeight: "700",
    marginBottom: 12,
    color: "#1B2559"
  },
  subtitle: {
    fontSize: 16,
    color: "#667085",
    marginBottom: 24
  },
  form: {
    backgroundColor: "#ffffff",
    borderRadius: 16,
    padding: 20,
    shadowColor: "#101828",
    shadowOpacity: 0.08,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 8 },
    elevation: 3
  },
  label: {
    fontSize: 14,
    fontWeight: "600",
    marginBottom: 8,
    color: "#344054"
  },
  input: {
    borderWidth: 1,
    borderColor: "#D0D5DD",
    borderRadius: 12,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 16,
    marginBottom: 16
  },
  tierRow: {
    flexDirection: "row",
    gap: 12,
    marginBottom: 20
  },
  tierButton: {
    flex: 1,
    borderWidth: 1,
    borderColor: "#D0D5DD",
    borderRadius: 12,
    paddingVertical: 12,
    alignItems: "center"
  },
  tierSelected: {
    backgroundColor: "#4C6FFF",
    borderColor: "#4C6FFF"
  },
  tierText: {
    color: "#344054",
    fontWeight: "600"
  },
  tierSelectedText: {
    color: "#ffffff",
    fontWeight: "700"
  },
  submit: {
    backgroundColor: "#4C6FFF",
    borderRadius: 12,
    paddingVertical: 14,
    alignItems: "center"
  },
  submitText: {
    color: "#ffffff",
    fontSize: 16,
    fontWeight: "700"
  }
});

export default LoginScreen;
