import { ApolloServer } from "@apollo/server";
import { expressMiddleware } from "@apollo/server/express4";
import cors from "cors";
import dotenv from "dotenv";
import express, { Request, Response } from "express";

dotenv.config();

const PORT = process.env.GATEWAY_PORT ?? "4000";

type ServiceKey = "timeline" | "label" | "socialFeed";

const serviceEndpoints: Record<ServiceKey, string> = {
  timeline: process.env.TIMELINE_SERVICE_URL ?? "http://timeline:7000",
  label: process.env.LABEL_SERVICE_URL ?? "http://label:7000",
  socialFeed: process.env.SOCIAL_FEED_SERVICE_URL ?? "http://social-feed:7000"
};

const typeDefs = /* GraphQL */ `
  type Health {
    status: String!
    service: String!
    time: String!
  }

  type TimelineEntry {
    category: String!
    started_at: String!
    ended_at: String!
    confidence: Float!
    source: String!
  }

  type Label {
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
  }

  type Query {
    health: Health!
    timeline(userId: ID!): [TimelineEntry!]!
    labels(userId: ID!): [Label!]!
    feed(userId: ID!): [FeedItem!]!
  }
`;

const resolvers = {
  Query: {
    health: async () => ({
      status: "ok",
      service: "gateway",
      time: new Date().toISOString()
    }),
    timeline: async (_: unknown, args: { userId: string }) => {
      const url = `${serviceEndpoints.timeline}/v1/timeline/${args.userId}`;
      return fetchJSON(url);
    },
    labels: async (_: unknown, args: { userId: string }) => {
      const url = `${serviceEndpoints.label}/v1/labels/${args.userId}`;
      return fetchJSON(url);
    },
    feed: async (_: unknown, args: { userId: string }) => {
      const url = `${serviceEndpoints.socialFeed}/v1/feed/${args.userId}`;
      return fetchJSON(url);
    }
  }
};

async function fetchJSON(url: string) {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to fetch ${url}: ${response.status}`);
  }
  return response.json();
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
    expressMiddleware(server)
  );

  app.listen(PORT, () => {
    console.log(`🚀 Gateway ready at http://localhost:${PORT}/graphql`);
  });
}

bootstrap().catch((err) => {
  console.error("Failed to start gateway", err);
  process.exit(1);
});
