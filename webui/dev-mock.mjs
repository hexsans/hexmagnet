import { spawn } from "child_process";
import { fileURLToPath } from "url";
import { dirname, resolve } from "path";

const __dirname = dirname(fileURLToPath(import.meta.url));

const MOCK_PORT = process.env.MOCK_PORT || 4000;
const VITE_PORT = process.env.VITE_PORT || 3000;

const mock = spawn("node", [resolve(__dirname, "mock-server.mjs")], {
  stdio: "inherit",
  env: { ...process.env, MOCK_PORT: String(MOCK_PORT) },
});

const vite = spawn("npx", ["vite", "--port", String(VITE_PORT)], {
  stdio: "inherit",
  cwd: __dirname,
  env: {
    ...process.env,
    VITE_GRAPHQL_URL: `http://localhost:${MOCK_PORT}/graphql`,
  },
});

function cleanup() {
  mock.kill("SIGTERM");
  vite.kill("SIGTERM");
  process.exit();
}

process.on("SIGINT", cleanup);
process.on("SIGTERM", cleanup);

mock.on("exit", (code) => {
  if (code !== 0)
    console.error(`[dev-mock] Mock server exited with code ${code}`);
  vite.kill("SIGTERM");
  process.exit(code);
});

vite.on("exit", (code) => {
  mock.kill("SIGTERM");
  process.exit(code);
});
