import { AuthHeaders } from "../types/auth";

const GATEWAY_URL =
  process.env.EXPO_PUBLIC_GATEWAY_URL ?? "http://localhost:4000/graphql";

interface GraphQLRequest {
  query: string;
  variables?: Record<string, unknown>;
}

export async function graphqlFetch<T>(
  body: GraphQLRequest,
  headers: AuthHeaders
): Promise<T> {
  const response = await fetch(GATEWAY_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "x-user-id": headers.userId,
      "x-user-tier": headers.tier
    },
    body: JSON.stringify(body)
  });

  const json = await response.json();
  if (!response.ok || json.errors) {
    const message =
      json.errors?.[0]?.message ??
      `GraphQL request failed with status ${response.status}`;
    throw new Error(message);
  }
  return json.data as T;
}
