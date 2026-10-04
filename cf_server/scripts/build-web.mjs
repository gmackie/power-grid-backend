import { execFileSync } from "node:child_process"
import { cpSync, existsSync, rmSync } from "node:fs"
import { resolve } from "node:path"

const client = process.argv[2]
if (!client || !existsSync(resolve(client, "package.json"))) {
  throw new Error("Usage: node scripts/build-web.mjs /path/to/react_client")
}
execFileSync("npm", ["run", "build"], {
  cwd: resolve(client), stdio: "inherit",
  env: { ...process.env, VITE_WS_URL: "wss://power.gmac.io/ws" }
})
rmSync("web-dist", { recursive: true, force: true })
cpSync(resolve(client, "dist"), "web-dist", { recursive: true })
