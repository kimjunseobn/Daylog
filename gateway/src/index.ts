import { ApolloServer } from "@apollo/server";
import { expressMiddleware } from "@apollo/server/express4";
import cors from "cors";
import express, { Request, Response } from "express";
import { GraphQLError } from "graphql";
import GraphQLJSON from "graphql-type-json";

import { loadConfig } from "./config";

const { port: PORT, endpoints } = loadConfig();

type GraphQLContext = {
  userId?: string;
  tier?: string;
};

const typeDefs = /* GraphQL */ `
  scalar JSON

  type Health {
    status: String!
    service: String!
    time: String!
  }

  type TimelineEntry {
    event_id: ID!
    user_id: ID!
    category: String!
    started_at: String!
    ended_at: String!
    confidence: Float!
    source: String!
    geo_context: JSON
    metadata: JSON
    source_event_ids: [ID!]!
  }

  type Label {
    id: ID!
    user_id: ID!
    label_key: String!
    label_value: String!
    is_verified: Boolean!
    verified_at: String
    last_updated: String!
  }

  type FeedItem {
    post_id: String!
    user_id: String!
    timeline_id: String!
    category: String!
    message: String!
    created_at: String!
    metadata: JSON
  }

  type Entitlement {
    user_id: ID!
    tier: String!
    status: String!
    renewal_date: String
    stripe_subscription_id: String
  }

  input UpsertLabelInput {
    user_id: ID!
    label_key: String!
    label_value: String!
    is_verified: Boolean
    verified_at: String
  }

  type MutationPayload {
    success: Boolean!
    message: String
  }

  type Community {
    id: ID!
    access_level: String!
    title: String!
    description: String!
    is_pro_only: Boolean!
    created_at: String!
  }

  type Membership {
    community_id: ID!
    user_id: ID!
    role: String!
    joined_at: String!
  }

  input CreateCommunityInput {
    access_level: String
    title: String!
    description: String
    is_pro_only: Boolean
  }

  input JoinCommunityInput {
    community_id: ID!
    user_id: ID!
    role: String
  }

  input CreateFeedPostInput {
    user_id: ID!
    timeline_id: ID!
    category: String!
    message: String!
    metadata: JSON
  }

  type Query {
    health: Health!
    timeline(userId: ID!, limit: Int): [TimelineEntry!]!
    labels(userId: ID!): [Label!]!
    feed(userId: ID!, limit: Int): [FeedItem!]!
    communities(includePro: Boolean): [Community!]!
    entitlement(userId: ID!): Entitlement
    viewerEntitlement: Entitlement
  }

  type Mutation {
    upsertLabel(input: UpsertLabelInput!): Label!
    createFeedPost(input: CreateFeedPostInput!): FeedItem!
    createCommunity(input: CreateCommunityInput!): Community!
    joinCommunity(input: JoinCommunityInput!): Membership!
  }
`;

const resolvers = {
  JSON: GraphQLJSON,
  Query: {
    health: async () => ({
      status: "ok",
      service: "gateway",
      time: new Date().toISOString()
    }),
    timeline: async (_: unknown, args: { userId: string; limit?: number }) => {
      const search = new URLSearchParams();
      if (args.limit) {
        search.set("limit", String(args.limit));
      }
      const url = `${endpoints.timeline}/v1/timeline/${args.userId}${
        search.size ? `?${search}` : ""
      }`;
      return fetchJSON(url);
    },
    labels: async (_: unknown, args: { userId: string }) => {
      const url = `${endpoints.label}/v1/labels/${args.userId}`;
      return fetchJSON(url);
    },
    feed: async (_: unknown, args: { userId: string; limit?: number }, ctx: GraphQLContext) => {
      if (args.userId !== ctx.userId) {
        requireTier(ctx, "pro"); // allow viewing others only for pro
      }
      const params = new URLSearchParams();
      if (args.limit) {
        params.set("limit", String(args.limit));
      }
      const url = `${endpoints.socialFeed}/v1/feed/${args.userId}${
        params.size ? `?${params}` : ""
      }`;
      return fetchJSON(url);
    },
    communities: async (_: unknown, args: { includePro?: boolean }, ctx: GraphQLContext) => {
      const params = new URLSearchParams();
      if (args.includePro) {
        requireTier(ctx, "pro");
        params.set("include_pro", "true");
      }
      const url = `${endpoints.community}/v1/communities${
        params.size ? `?${params}` : ""
      }`;
      return fetchJSON(url);
    },
    entitlement: async (_: unknown, args: { userId: string }) => {
      const url = `${endpoints.billing}/v1/entitlements/${args.userId}`;
      return fetchJSON(url);
    },
    viewerEntitlement: async (_: unknown, __: unknown, ctx: GraphQLContext) => {
      const userId = requireAuthenticated(ctx);
      const url = `${endpoints.billing}/v1/entitlements/${userId}`;
      return fetchJSON(url);
    }
  },
  Mutation: {
    upsertLabel: async (_: unknown, args: { input: Record<string, unknown> }, ctx: GraphQLContext) => {
      const userId = requireAuthenticated(ctx);
      if (typeof args.input.user_id !== "string" || args.input.user_id.length === 0) {
        args.input.user_id = userId;
      }
      if (args.input.user_id !== userId) {
        throw new GraphQLError("cannot modify another user's labels", {
          extensions: { code: "FORBIDDEN" }
        });
      }
      const url = `${endpoints.label}/v1/labels`;
      return fetchJSON(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify(args.input)
      });
    },
    createFeedPost: async (_: unknown, args: { input: Record<string, unknown> }, ctx: GraphQLContext) => {
      const userId = requireAuthenticated(ctx);
      if (typeof args.input.user_id !== "string" || args.input.user_id.length === 0) {
        args.input.user_id = userId;
      }
      if (args.input.user_id !== userId) {
        throw new GraphQLError("cannot create feed post for another user", {
          extensions: { code: "FORBIDDEN" }
        });
      }
      const url = `${endpoints.socialFeed}/v1/feed`;
      return fetchJSON(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(args.input)
      });
    },
    createCommunity: async (_: unknown, args: { input: Record<string, unknown> }, ctx: GraphQLContext) => {
      requireAuthenticated(ctx);
      const url = `${endpoints.community}/v1/communities`;
      return fetchJSON(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(args.input)
      });
    },
    joinCommunity: async (_: unknown, args: { input: Record<string, unknown> }, ctx: GraphQLContext) => {
      const userId = requireAuthenticated(ctx);
      const communityId = args.input["community_id"];
      if (typeof communityId !== "string" || communityId.length === 0) {
        throw new GraphQLError("community_id is required", {
          extensions: { code: "BAD_USER_INPUT" }
        });
      }
      if (typeof args.input.user_id !== "string" || args.input.user_id.length === 0) {
        args.input.user_id = userId;
      }
      if (args.input.user_id !== userId) {
        throw new GraphQLError("cannot join community on behalf of another user", {
          extensions: { code: "FORBIDDEN" }
        });
      }
      const communityId = args.input["community_id"];
      const { community_id, ...rest } = args.input;
      const url = `${endpoints.community}/v1/communities/${community_id}/join`;
      return fetchJSON(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(rest)
      });
    }
  }
};

async function fetchJSON(url: string, options?: RequestInit) {
  try {
    const response = await fetch(url, options);
    const text = await response.text();
    if (!response.ok) {
      throw new GraphQLError(`Upstream request failed`, {
        extensions: {
          code: "UPSTREAM_ERROR",
          http: { status: response.status },
          detail: text
        }
      });
    }
    return text.length ? JSON.parse(text) : null;
  } catch (error) {
    if (error instanceof GraphQLError) {
      throw error;
    }
    throw new GraphQLError("Failed to contact upstream service", {
      extensions: {
        code: "UPSTREAM_NETWORK_ERROR",
        detail: (error as Error).message
      }
    });
  }
}

async function bootstrap() {
  const app = express();
  const server = new ApolloServer({
    typeDefs,
    resolvers
  });

  await server.start();

  app.get("/healthz", (_req: Request, res: Response) => {
    res.json({ status: "ok", service: "gateway", time: new Date().toISOString() });
  });

  app.use(
    "/graphql",
    cors<cors.CorsRequest>(),
    express.json(),
    expressMiddleware(server, {
      context: async ({ req }): Promise<GraphQLContext> => ({
        userId: req.header("x-user-id") ?? undefined,
        tier: req.header("x-user-tier") ?? undefined
      })
    })
  );

  app.listen(PORT, () => {
    console.log(`🚀 Gateway ready at http://localhost:${PORT}/graphql`);
  });
}

bootstrap().catch((err) => {
  console.error("Failed to start gateway", err);
  process.exit(1);
});

function requireAuthenticated(ctx: GraphQLContext): string {
  if (!ctx.userId) {
    throw new GraphQLError("authentication required", {
      extensions: { code: "UNAUTHENTICATED" }
    });
  }
  return ctx.userId;
}

function requireTier(ctx: GraphQLContext, required: string) {
  if (!ctx.tier) {
    throw new GraphQLError("insufficient permissions", {
      extensions: { code: "FORBIDDEN" }
    });
  }
  if (tierRank(ctx.tier) < tierRank(required)) {
    throw new GraphQLError(`tier ${required} required`, {
      extensions: { code: "FORBIDDEN" }
    });
  }
}

function tierRank(tier: string): number {
  switch (tier.toLowerCase()) {
    case "pro":
      return 2;
    case "free":
      return 1;
    default:
      return 0;
  }
}
