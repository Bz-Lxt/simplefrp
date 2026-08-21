import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: ".",
  timeout: 30_000,
  use: {
    baseURL: process.env.VISITOR_URL || "http://127.0.0.1:42817",
  },
});
