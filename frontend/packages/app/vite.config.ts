/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: { port: 5173 },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    env: {
      VITE_API_URL: "http://api.test",
      VITE_SUPABASE_URL: "http://supabase.test",
      VITE_SUPABASE_ANON_KEY: "anon",
    },
  },
});
