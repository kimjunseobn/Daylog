import dotenv from "dotenv";

dotenv.config();

export interface ServiceEndpoints {
  timeline: string;
  label: string;
  socialFeed: string;
  ingestion: string;
  community: string;
  billing: string;
}

export interface GatewayConfig {
  port: string;
  endpoints: ServiceEndpoints;
}

function env(name: string, fallback: string): string {
  return process.env[name] && process.env[name]!.length > 0
    ? process.env[name]!
    : fallback;
}

export function loadConfig(): GatewayConfig {
  return {
    port: env("GATEWAY_PORT", "4000"),
    endpoints: {
      timeline: env("TIMELINE_SERVICE_URL", "http://localhost:7002"),
      label: env("LABEL_SERVICE_URL", "http://localhost:7003"),
      socialFeed: env("SOCIAL_FEED_SERVICE_URL", "http://localhost:7004"),
      ingestion: env("INGESTION_SERVICE_URL", "http://localhost:7001"),
      community: env("COMMUNITY_SERVICE_URL", "http://localhost:7005"),
      billing: env("BILLING_SERVICE_URL", "http://localhost:7006")
    }
  };
}
